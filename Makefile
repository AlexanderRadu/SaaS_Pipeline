.PHONY: secrets, generate-api


secrets:
	@cp .env.example .env
	@auth_pw=$$(openssl rand -hex 32); \
     status_pw=$$(openssl rand -hex 32); \
     catalog_pw=$$(openssl rand -hex 32); \
     rabbit_pw=$$(openssl rand -hex 32); \
     redis_pw=$$(openssl rand -hex 32); \
     jwt_secret=$$(openssl rand -hex 32); \
     internal_api_token=$$(openssl rand -hex 32); \
     sed -i.bak \
     -e "s/<set-me-auth>/$$auth_pw/g" \
     		-e "s/<set-me-status>/$$status_pw/g" \
     		-e "s/<set-me-catalog>/$$catalog_pw/g" \
     		-e "s/<set-me-rabbit>/$$rabbit_pw/g" \
     		-e "s/<set-me-redis>/$$redis_pw/g" \
     		-e "s/<generate-256-bit-secret>/$$jwt_secret/g" \
     		-e "s/<generate-256-bit-token>/$$internal_api_token/g" \
     		.env; \
    rm -f .env.bak


generate-api:
	oapi-codegen -config api/gen/auth-cfg.yaml api/openapi.yaml
	oapi-codegen -config api/gen/catalog-cfg.yaml api/openapi.yaml
	oapi-codegen -config api/gen/deploy-cfg.yaml api/openapi.yaml
	oapi-codegen -config api/gen/status-cfg.yaml api/openapi.yaml