package main

import (
	"context"
	"fmt"
	"github.com/bufbuild/connect-go"
	"log"
	"net/http"

	couponv1 "coupon/gen/coupon/v1"
	"coupon/gen/coupon/v1/couponv1connect"
)

func main() {
	client := couponv1connect.NewCouponServiceClient(
		http.DefaultClient,
		"http://localhost:8080",
	)

	ctx := context.Background()

	//startTs := time.Now().Unix()
	//createResp, err := client.CreateCampaign(
	//	ctx,
	//	connect.NewRequest(&couponv1.CreateCampaignRequest{
	//		Name:           "test campaign",
	//		StartTimestamp: startTs,
	//		TotalCoupons:   10,
	//	}),
	//)
	//if err != nil {
	//	log.Fatalf("CreateCampaign failed: %v", err)
	//}
	//campaignID := createResp.Msg.CampaignId
	campaignID := "1751447818149810800"
	fmt.Printf("Created campaign: %s\n", campaignID)

	issueResp, err := client.IssueCoupon(
		ctx,
		connect.NewRequest(&couponv1.IssueCouponRequest{
			CampaignId: campaignID,
		}),
	)
	if err != nil {
		log.Fatalf("IssueCoupon failed: %v", err)
	}
	fmt.Printf("Issued coupon: %s\n", issueResp.Msg.CouponCode)

	getResp, err := client.GetCampaign(
		ctx,
		connect.NewRequest(&couponv1.GetCampaignRequest{
			CampaignId: campaignID,
		}),
	)
	if err != nil {
		log.Fatalf("GetCampaign failed: %v", err)
	}
	fmt.Printf("Campaign Name=%s, Issued=%v\n",
		getResp.Msg.Name, getResp.Msg.IssuedCoupons)
}
