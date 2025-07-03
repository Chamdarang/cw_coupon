package main

import (
	"log"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"

	"coupon/gen/coupon/v1/couponv1connect"
	"coupon/internal/server"
	"coupon/internal/store"
)

func main() {
	// Redis 초기화
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	redisStore := store.NewRedisStore(rdb)

	// DB 초기화
	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/cw_coupon?parseTime=true")
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("DB ping failed: %v", err)
	}

	dbStore := store.NewDBStore(db)

	// CouponServer 생성
	couponServer := server.NewCouponServer(redisStore, dbStore)

	mux := http.NewServeMux()
	path, handler := couponv1connect.NewCouponServiceHandler(couponServer)
	mux.Handle(path, handler)

	addr := ":8080"
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(
		addr,
		h2c.NewHandler(mux, &http2.Server{}),
	); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
