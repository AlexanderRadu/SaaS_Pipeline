.PHONY: secrets

secrets:
	@cp .env.example .env
	@auth_pw=$$(openssl rand -hex 32); \
     status_pw=$$(openssl rand -hex 32); \
     catalog_pw=$$(openssl rand -hex 32); \
     rabbit_pw=$$(openssl rand -hex 32); \
     redis_pw=$$(openssl rand -hex 32); \
     jwt_secret=$$(openssl rand -hex 32); \
     sed -i.bak \
     -e "s/<set-me-auth>/$$auth_pw/g" \
     		-e "s/<set-me-status>/$$status_pw/g" \
     		-e "s/<set-me-catalog>/$$catalog_pw/g" \
     		-e "s/<set-me-rabbit>/$$rabbit_pw/g" \
     		-e "s/<set-me-redis>/$$redis_pw/g" \
     		-e "s/<generate-256-bit-secret>/$$jwt_secret/g" \
     		.env; \
    rm -f .env.bak


