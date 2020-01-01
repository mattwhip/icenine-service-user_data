# Service config
SERVICE_NAME=user_data
SERVICE_VERS=latest
GC_PROJECT=indian-game-system
GC_REGION=us-central
GC_ZONE=us-central1-a

###########
# Docker
###########

# Build the service
build:
	# Build Docker image
	docker build -t gcr.io/${GC_PROJECT}/${SERVICE_NAME}:${SERVICE_VERS} -f Dockerfile . --build-arg GC_PROJECT=${GC_PROJECT}

# Push the service image
push:
	docker push gcr.io/${GC_PROJECT}/${SERVICE_NAME}:${SERVICE_VERS}

###########
# Migrate
###########
migrate:
	echo "Migrating ${SERVICE_NAME} db"
	buffalo db migrate

###########
# Seed
###########
seed:
	echo "Seeding ${SERVICE_NAME} db"
	buffalo t db:seed
