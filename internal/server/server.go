package server

import (
	"context"
	couponv1 "coupon/gen/coupon/v1"
	"coupon/gen/coupon/v1/couponv1connect"
	"coupon/internal/store"
	"fmt"
	"github.com/bufbuild/connect-go"
	"log"
	"math/rand"
	"time"
)

type CouponServer struct {
	redis *store.RedisStore
	db    *store.DBStore
}

func NewCouponServer(redis *store.RedisStore, db *store.DBStore) *CouponServer {
	return &CouponServer{
		redis: redis,
		db:    db,
	}
}

// 기능들 구현됐는지(쿠폰서버로 사용가능한지) 확인용
var _ couponv1connect.CouponServiceHandler = (*CouponServer)(nil)

func (s *CouponServer) CreateCampaign(
	originCtx context.Context,
	req *connect.Request[couponv1.CreateCampaignRequest],
) (*connect.Response[couponv1.CreateCampaignResponse], error) {
	ctx, cancel := context.WithTimeout(originCtx, 2*time.Second)
	defer cancel()
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	err := s.db.CreateCampaign(ctx, id, req.Msg.Name, req.Msg.StartTimestamp, int(req.Msg.TotalCoupons))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create campaign: %w", err))
	}
	return connect.NewResponse(&couponv1.CreateCampaignResponse{CampaignId: id}), nil
}

func (s *CouponServer) GetCampaign(
	originCtx context.Context,
	req *connect.Request[couponv1.GetCampaignRequest],
) (*connect.Response[couponv1.GetCampaignResponse], error) {
	ctx, cancel := context.WithTimeout(originCtx, 1*time.Second)
	defer cancel()
	campaign, coupons, err := s.db.GetCampaignWithCoupons(ctx, req.Msg.CampaignId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("campaign not found: %w", err))
	}

	return connect.NewResponse(&couponv1.GetCampaignResponse{
		Name:           campaign.Name,
		StartTimestamp: campaign.StartTimestamp,
		TotalCoupons:   int32(campaign.TotalCoupons),
		IssuedCoupons:  coupons,
	}), nil
}

func (s *CouponServer) IssueCoupon(
	originCtx context.Context,
	req *connect.Request[couponv1.IssueCouponRequest],
) (*connect.Response[couponv1.IssueCouponResponse], error) {
	ctx, cancel := context.WithTimeout(originCtx, 3*time.Second)
	defer cancel()

	campaign, err := s.db.GetCampaign(ctx, req.Msg.CampaignId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("campaign not found: %w", err))
	}

	if time.Now().Unix() < campaign.StartTimestamp {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("campaign not started"))
	}

	_, err = s.redis.TryIssueCoupon(ctx, req.Msg.CampaignId, campaign.TotalCoupons)
	if err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, err)
	}

	code := generateCouponCode()

	if err := s.db.SaveCoupon(ctx, req.Msg.CampaignId, code); err != nil {
		// DB 저장 실패시 Redis 롤백
		if rbErr := s.redis.RollbackIssueCoupon(ctx, req.Msg.CampaignId); rbErr != nil {
			//롤백도 실패하면 로그 남김
			log.Printf("rollback failed for campaign %s: %v", req.Msg.CampaignId, rbErr)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save coupon: %w", err))
	}

	return connect.NewResponse(&couponv1.IssueCouponResponse{CouponCode: code}), nil
}

func generateCouponCode() string {
	charset := []rune("가나다라마바사아자차카타파하거너더러머버서어저처커터퍼허0123456789")
	length := rand.Intn(5) + 5 // ~10자
	b := make([]rune, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
