## 쿠폰발행 시스템

캠페인 생성, 캠페인 조회, 쿠폰 발행 3개 기능을 구현  
- 캠페인 생성과 조회는 MySQL에서만 처리
- 쿠폰 발행은 Redis를 추가로 사용
    - Redis를 발급 수량을 관리하는 atomic counter로 사용
    - DB를 직접 조회하여 발급 수량을 계산하는 것보다 훨씬 빠름


## 쿠폰 발행 처리 흐름

- **캠페인 존재 및 시작 시간 확인**  
  - MySQL에서 캠페인 정보 조회  
  - 캠페인 존재 여부, 시작 시간 조건 체크

- **발급 수량 확인 및 카운트**
  - Redis Lua 스크립트로 atomic하게 수량 체크 및 INCR
  - 초과 시 발급 차단

- **쿠폰 코드 생성 및 DB 등록**
  - 새로운 쿠폰 코드 생성  
  -  MySQL에 발급 내역 저장


## 설계 선택 이유 

- **Redis에 쿠폰 Pool을 사전에 채우는 방식은 사용하지 않음**
    - 캠페인 생성 시 대량 쿠폰 사전 생성 필요 → 부하 급증
    - Redis 휘발성 특성상 Pool 유실 시 재생성 부담
    - Pool의 중복 방지를 위해 DB에도 미리 발급 목록을 저장해야 하며, 이 경우 DB 부하도 커짐

- **캠페인 존재·시작 여부는 Redis에 캐싱하지 않음**
    - 캠페인 설정 변경 시 Redis-DB 간 sync 문제 발생 가능
    - 캠페인 상태는 DB에서만 확인
    - Redis는 오직 발급 수량 atomic control 용도로 한정


---
## 개발환경

- **언어**: Go 1.23.2
- **DB**: MySQL
- **카운터/캐시**: Redis
- **서버 실행 파일**: `./cmd/server/main.go`
- **테스트 클라이언트**
    - `./cmd/client/client.go` (기능 작동 테스트)
    - `./cmd/client/client_mul.go` (동시성 테스트)

## DB 스키마

### campaigns

| 컬럼명        | 타입         | 길이 | 설명 |
|--------------|-------------|-------|-------|
| id           | VARCHAR      | 64    | 캠페인 ID (PK) |
| name         | VARCHAR      | 255   | 캠페인 이름 |
| start_time   | DATETIME     | -     | 캠페인 시작 시각 |
| total_coupons| INT          | -     | 캠페인 총 발급 수량 |



### coupons

| 컬럼명       | 타입        | 길이 | 설명 |
|-------------|------------|-------|-------|
| id          | BIGINT      | -     | 쿠폰 ID (PK, AUTO_INCREMENT) |
| campaign_id | VARCHAR     | 64    | 캠페인 ID |
| coupon_code | VARCHAR     | 64    | 쿠폰 코드 |
| issued_at   | DATETIME    | -     | 발급 시각 |

- `coupon_code`는 `campaign_id`와 함께 UNIQUE 설정
  - 쿠폰 코드 단독으로 UNIQUE 처리하면 캠페인이 늘어날수록 코드 관리와 생성의 난이도가 급격히 상승
  - `campaign_id, coupon_code` 조합 인덱스로 인해 캠페인별 쿠폰 코드 조회 시 인덱스 탐색 비용이 낮음


## 동시성 및 수평 확장 설계

- **동시성 처리**
  - Redis Lua 스크립트를 통해 쿠폰 수량 체크 + INCR을 atomic하게 처리하여 race condition 방지
  - 클라이언트 부하 테스트(client_mul.go)로 동시성 상황 시뮬레이션

- **현재 수평 확장 대응**
  - 서버는 stateless 구조로 여러 인스턴스 동시 운영 가능
  - Redis atomic counter로 race-free 상태 관리
  - DB는 단순 기록용으로 insert 충돌 최소화

- **추가 수평 확장을 위한 고려**
  - Redis Cluster 구성
  - DB 샤딩 또는 파티셔닝
  - 서버 오토스케일링
