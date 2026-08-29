# instance-controller

AWS Lambda에서 실행되는 단일 EC2 인스턴스 시작/중지 API입니다. Gin, Swagger UI, AWS SDK for Go v2, AWS Lambda Web Adapter를 사용합니다.

## API

- `GET /api/v1/instance`: 상태와 public IPv4/IPv6 조회
- `POST /api/v1/instance/state`: `{"action":"start"}` 또는 `{"action":"stop"}`
- `GET /swagger/index.html`: Swagger UI (인증 제외)
- `GET /healthz`: Lambda Web Adapter readiness check (인증 제외)

EC2 API에는 HTTP Basic Authentication이 적용됩니다. Swagger UI와 readiness check는 인증에서 제외됩니다.
모든 origin에 대해 CORS가 허용되며 `Authorization` 및 `Content-Type` 요청 헤더를 사용할 수 있습니다.

## 환경 변수

| 이름 | 설명 |
| --- | --- |
| `EC2_REGION` | 대상 EC2 리전 (없으면 Lambda가 제공하는 `AWS_REGION` 사용) |
| `EC2_INSTANCE_ID` | 대상 EC2 인스턴스 ID |
| `BASIC_AUTH_USERNAME` | HTTP Basic Auth 사용자명 |
| `BASIC_AUTH_PASSWORD` | HTTP Basic Auth 비밀번호 |
| `PORT` | HTTP 포트, 기본값 `8080` |

AWS 자격증명은 SDK 기본 credential chain을 사용합니다. Lambda에서는 실행 역할이 자동으로 사용됩니다.

## 로컬 실행

```bash
cp .env.example .env
set -a; source .env; set +a
go run ./cmd/server
```

```bash
curl -u admin:password http://localhost:8080/api/v1/instance
curl -u admin:password \
  -H 'Content-Type: application/json' \
  -d '{"action":"start"}' \
  http://localhost:8080/api/v1/instance/state
```

로컬 AWS 인증에는 AWS CLI 프로파일 또는 `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` 등을 사용할 수 있습니다.

## Lambda 배포

Docker와 AWS SAM CLI가 필요합니다.

```bash
sam build
sam deploy --guided
```

템플릿은 Lambda Function URL을 생성하며 애플리케이션 계층에서 Basic Auth를 검증합니다. 운영에서는 Basic Auth 비밀번호를 평문 파라미터 대신 Secrets Manager 동적 참조 등으로 주입하는 것을 권장합니다.

필요한 실행 역할 권한은 `ec2:DescribeInstances`, `ec2:StartInstances`, `ec2:StopInstances`입니다. `DescribeInstances`는 AWS에서 리소스 수준 권한을 지원하지 않아 정책의 `Resource`가 `*`입니다.

## Swagger 재생성

```bash
make swagger
```
