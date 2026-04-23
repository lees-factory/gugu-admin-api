# Gugu Admin API

이 저장소는 어드민 배치, 상품 메타 동기화, AliExpress OAuth 토큰 관리를 위한 API를 제공합니다.

## AliExpress OAuth API 흐름 (DROPSHIPPING/AFFILIATE)

핵심: `code`는 callback 호출로 생성되지 않습니다.  
AliExpress 인가 페이지에서 판매자가 로그인/승인한 뒤 callback으로 전달됩니다.

### 1. 관리자 로그인 (Bearer 토큰 발급)

`POST /v1/admin/auth/login`

필수 헤더:
- `Content-Type: application/json`

예시:

```bash
curl -sS -X POST "https://lees-admin.duckdns.org/v1/admin/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"id":"<admin-login-id>","password":"<admin-password>"}'
```

### 2. 인가 URL 생성 요청

`GET /v1/aliexpress/oauth/authorize-url?app_type=DROPSHIPPING`

필수 인증 헤더(둘 중 하나):
- `Authorization: Bearer <admin-access-token>`
- `X-Admin-API-Key: <admin-api-key>`

예시:

```bash
curl -sS "https://lees-admin.duckdns.org/v1/aliexpress/oauth/authorize-url?app_type=DROPSHIPPING" \
  -H "Authorization: Bearer <admin-access-token>"
```

응답에는 아래 값이 포함됩니다.
- `authorization_url`: AliExpress 로그인/승인 페이지 URL
- `callback_url`: 우리 서버 callback 엔드포인트

### 3. `authorization_url` 접속 후 판매자 승인

판매자가 AliExpress에서 로그인/승인을 완료하면 callback URL로 리다이렉트됩니다.

```text
.../v1/aliexpress/oauth/callback/dropshipping?code=<authorization_code>
```

AFFILIATE는 아래 callback을 사용합니다.

```text
.../v1/aliexpress/oauth/callback/affiliate?code=<authorization_code>
```

### 4. callback에서 code 교환 및 토큰 저장

callback 엔드포인트:
- `GET /v1/aliexpress/oauth/callback/dropshipping`
- `GET /v1/aliexpress/oauth/callback/affiliate`

필수 쿼리:
- `code` (AliExpress가 발급한 인가 코드)

주의사항:
- callback은 AliExpress 리다이렉트로 호출되므로 어드민 인증 헤더가 필요 없습니다.
- 인가 코드는 수명이 짧습니다(가이드 기준 약 30분).
- 재사용/만료/오류 code는 `InvalidCode` 같은 에러가 반환됩니다.

### 5. 토큰 상태 확인

`GET /v1/aliexpress/token/status?app_type=DROPSHIPPING`

필수 인증 헤더(둘 중 하나):
- `Authorization: Bearer <admin-access-token>`
- `X-Admin-API-Key: <admin-api-key>`

예시:

```bash
curl -sS "https://lees-admin.duckdns.org/v1/aliexpress/token/status?app_type=DROPSHIPPING" \
  -H "Authorization: Bearer <admin-access-token>"
```

정상 반영 시 `status=ACTIVE`이며 만료 시각이 최신으로 갱신됩니다.
