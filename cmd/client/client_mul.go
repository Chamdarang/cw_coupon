package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/bufbuild/connect-go"
	"log"
	"net/http"
	"sync"
	"time"

	couponv1 "coupon/gen/coupon/v1"
	"coupon/gen/coupon/v1/couponv1connect"
)

func main() {
	client := couponv1connect.NewCouponServiceClient(http.DefaultClient, "http://localhost:8080")
	ctx := context.Background()

	// 캠페인 생성
	createResp, err := client.CreateCampaign(ctx, connect.NewRequest(&couponv1.CreateCampaignRequest{
		Name:           "test",
		StartTimestamp: time.Now().Add(2 * time.Second).Unix(),
		TotalCoupons:   20000,
	}))
	if err != nil {
		log.Fatalf("CreateCampaign failed: %v", err)
	}
	campaignID := createResp.Msg.CampaignId
	fmt.Printf("Created campaign: %s\n", campaignID)

	const totalRequests = 10000 //초당 약 1000개,
	const concurrency = 100     //너무 많으면 그냥 실패함

	var wg sync.WaitGroup
	jobs := make(chan int, totalRequests)

	// worker
	var connectErr *connect.Error
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				_, err := client.IssueCoupon(ctx, connect.NewRequest(&couponv1.IssueCouponRequest{
					CampaignId: campaignID,
				}))

				if err != nil {
					if errors.As(err, &connectErr) {
						switch connectErr.Code() {
						case connect.CodeResourceExhausted:
							//로깅
						} //이외 여러가지 추가
					}
					fmt.Printf("Fail, %v\n", err)
				}
			}
		}()
	}

	// rate limiter (약 1000 req/sec)
	ticker := time.NewTicker(time.Millisecond) // 1000 ms = 1s, 1000req/sec
	defer ticker.Stop()

	start := time.Now()

	for i := 0; i < totalRequests; i++ {
		<-ticker.C
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	duration := time.Since(start)
	fmt.Printf("Finish, %d requests in %v (%.2f req/sec)\n",
		totalRequests, duration, float64(totalRequests)/duration.Seconds())
}
