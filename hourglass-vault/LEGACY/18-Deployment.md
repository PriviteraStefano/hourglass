# Deployment & Infrastructure

Production deployment, Docker setup, and infrastructure considerations.

---

## Overview

Hourglass is deployed as:
- **Docker container** running Go backend
- **Frontend** as static files (served via CDN or reverse proxy)
- **PostgreSQL** database (managed separately)

---

## Docker Setup

### Building Docker Image

**File:** `Dockerfile` (multi-stage)

```dockerfile
# Stage 1: Build Go binary
FROM golang:1.26.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o hourglass ./cmd/server

# Stage 2: Runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/hourglass /app/
COPY migrations /app/migrations
EXPOSE 8080
CMD ["./hourglass"]
```

**Build command:**
```bash
docker build -t hourglass:latest .
```

### Docker Compose (Local Dev)

**File:** `docker-compose.yml`

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:13-alpine
    container_name: hourglass-postgres
    environment:
      POSTGRES_USER: hourglass
      POSTGRES_PASSWORD: hourglass
      POSTGRES_DB: hourglass
    ports:
      - \"5432:5432\"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: [\"CMD-SHELL\", \"pg_isready -U hourglass\"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

**Start:**
```bash
docker-compose up -d
```

---

## Environment Variables

### Backend

| Variable | Purpose | Example | Required |
|----------|---------|---------|----------|
| DATABASE_URL | PostgreSQL connection | postgres://user:pass@host:5432/db | Yes |
| JWT_SECRET | Token signing key | (32+ char random string) | Yes |
| PORT | HTTP server port | 8080 | No |
| ENVIRONMENT | Deployment stage | production, staging, development | No |
| LOG_LEVEL | Logging level | info, debug, error | No |

**Production Example:**
```bash
DATABASE_URL=postgres://hourglass:${DB_PASSWORD}@prod-postgres.internal:5432/hourglass?sslmode=require
JWT_SECRET=${RANDOM_32_CHAR_SECRET}
PORT=8080
ENVIRONMENT=production
LOG_LEVEL=info
```

### Frontend

| Variable | Purpose | Example |
|----------|---------|---------|
| VITE_API_URL | Backend base URL | https://api.hourglass.example.com |
| VITE_APP_NAME | App display name | Hourglass |

---

## Kubernetes Deployment

### Backend Deployment

**File:** `k8s/deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hourglass-backend
  namespace: hourglass
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hourglass-backend
  template:
    metadata:
      labels:
        app: hourglass-backend
    spec:
      containers:
      - name: hourglass
        image: hourglass:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: hourglass-secrets
              key: database-url
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: hourglass-secrets
              key: jwt-secret
        - name: ENVIRONMENT
          value: production
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: \"256Mi\"
            cpu: \"250m\"
          limits:
            memory: \"512Mi\"
            cpu: \"500m\"
---
apiVersion: v1
kind: Service
metadata:
  name: hourglass-backend
  namespace: hourglass
spec:
  selector:
    app: hourglass-backend
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

**Deploy:**
```bash
kubectl apply -f k8s/deployment.yaml
```

---

## Database Setup (Production)

### PostgreSQL Configuration

**Minimum Requirements:**
- Version: 13+
- 2 vCPU, 4GB RAM
- 50GB SSD storage

**Connection Pool Settings:**

In backend code (`internal/db/connection.go`):
```go
db.SetMaxOpenConns(25)      // Max concurrent connections
db.SetMaxIdleConns(5)       // Keep-alive connections
db.SetConnMaxLifetime(5 * time.Minute)  // Reuse connections
```

### Backup Strategy

**Daily backups:**
```bash
# Automated backup script
#!/bin/bash
BACKUP_FILE=/backups/hourglass_$(date +%Y%m%d_%H%M%S).sql.gz
pg_dump -h $DB_HOST -U hourglass -d hourglass | gzip > $BACKUP_FILE
aws s3 cp $BACKUP_FILE s3://hourglass-backups/
# Retain for 30 days
```

**Restore:**
```bash
gunzip < /backups/hourglass_YYYYMMDD_HHMMSS.sql.gz | psql -U hourglass -d hourglass
```

### Monitoring

Track:
- Connection count
- Query latency (p50, p95, p99)
- Disk usage
- Replication lag (if replicated)

---

## Frontend Deployment

### Static File Hosting

**Option 1: AWS S3 + CloudFront**

```bash
# Build frontend
cd web
npm run build

# Upload to S3
aws s3 sync dist/ s3://hourglass-frontend/ --delete

# Invalidate CloudFront cache
aws cloudfront create-invalidation --distribution-id XXXXX --paths \"/*\"
```

**Option 2: Nginx**

```nginx
server {
    listen 80;
    server_name hourglass.example.com;
    
    root /usr/share/nginx/html;
    index index.html;
    
    # Serve dist/ files
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    # Proxy API requests
    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Authorization $http_authorization;
    }
    
    # Cache busting for versioned assets
    location ~* \\.js$|\\.css$ {
        expires 1y;
        add_header Cache-Control \"public, immutable\";
    }
    
    # HTML should not be cached
    location ~* \\.html$ {
        expires -1;
        add_header Cache-Control \"no-cache, no-store, must-revalidate\";
    }
}
```

---

## SSL/TLS Setup

### Let's Encrypt (Free)

```bash
# Using Certbot with Nginx
sudo certbot certonly --nginx -d hourglass.example.com

# Renew automatically
sudo systemctl enable certbot.timer
sudo systemctl start certbot.timer
```

### Self-Signed (Testing)

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

---

## Health Checks

Backend exposes `/health` endpoint:

```
GET /health

Response:
{
  \"status\": \"ok\",
  \"database\": \"connected\",
  \"timestamp\": \"2024-04-01T10:00:00Z\"
}
```

Used by:
- Kubernetes liveness/readiness probes
- Load balancer health checks
- Monitoring systems

---

## Monitoring & Logging

### Application Logging

Backend logs to stdout (for containerization):

```go
log.Printf(\"[%s] %s\", level, message)
```

Captured by container orchestration:
- Docker: `docker logs`
- Kubernetes: `kubectl logs`
- Datadog/CloudWatch: Auto-collected

### Metrics to Track

- **API response times** (histogram)
- **Request rate** (counter)
- **Error rate** (counter)
- **Database query latency** (histogram)
- **Active connections** (gauge)
- **JWT validation failures** (counter)

### Alerting

Set up alerts for:
- 5XX error rate > 1%
- API latency p95 > 1 second
- Database connection pool nearly exhausted
- PostgreSQL disk usage > 80%

---

## Scaling Considerations

### Horizontal Scaling (Multiple Backends)

```yaml
# Kubernetes: Increase replicas
kubectl scale deployment hourglass-backend --replicas=5
```

**Stateless Design:**
- JWT tokens don't require session storage
- Queries scoped to organization
- Load balancer can route requests anywhere

### Database Scaling

**Read Replicas:**
- Primary handles writes
- Replicas handle reads
- Application connects to read replica for lists/queries

**Partitioning (Future):**
- Partition time_entries by organization_id
- Partition expenses by date
- Improves query performance on large datasets

---

## Security Checklist

### Before Production

- [ ] JWT_SECRET is 32+ random characters
- [ ] DATABASE_URL uses strong password
- [ ] DATABASE_URL enforces SSL/TLS (sslmode=require)
- [ ] All endpoints validate org membership
- [ ] CORS configured correctly
- [ ] Rate limiting on auth endpoints
- [ ] Password hashing verified (bcrypt)
- [ ] Refresh tokens expire appropriately
- [ ] API logs don't include sensitive data
- [ ] Database backups encrypted
- [ ] HTTPS enforced (redirect HTTP → HTTPS)

---

## Disaster Recovery

### RTO/RPO Targets

- **RTO** (Recovery Time Objective): < 4 hours
- **RPO** (Recovery Point Objective): < 1 hour

**Strategy:**
1. PostgreSQL automated backups every hour
2. S3 replication to secondary region
3. Frontend served from CDN (no data loss)
4. Database failover to replica (if available)

### Recovery Procedure

```bash
# 1. Restore database from latest backup
gunzip < backup.sql.gz | psql -U hourglass -d hourglass

# 2. Restart backend services
kubectl rollout restart deployment/hourglass-backend

# 3. Verify health
curl https://api.hourglass.example.com/health

# 4. Clear frontend cache
aws cloudfront create-invalidation --distribution-id XXXXX --paths \"/*\"
```

---

## Performance Optimization

### Database Indexes

Existing indexes (from schema):
```sql
CREATE INDEX idx_time_entries_org_id ON time_entries(organization_id);
CREATE INDEX idx_time_entries_user_id ON time_entries(user_id);
CREATE INDEX idx_time_entries_status ON time_entries(status);
```

Monitor slow queries:
```sql
-- PostgreSQL slow query log
log_min_duration_statement = 1000  -- Log queries > 1 second
```

### Frontend Caching

- **API responses:** React Query with 5-minute staleTime
- **Static assets:** Versioned, cached 1 year
- **HTML:** No cache, always fetch fresh

### Connection Pooling

- Backend uses sql.DB connection pool
- Max 25 concurrent connections
- Reuse connections > 5 minutes

---

## Cost Optimization

### AWS Example

- **EC2** (backend): t3.small = ~$15/month
- **RDS** (PostgreSQL): db.t3.micro = ~$20/month
- **S3** (frontend): < $1/month
- **CloudFront** (CDN): ~$5-10/month
- **Total**: ~$50/month for small deployment

### Cost Reduction

- Use shared database instance
- Serve frontend from S3 + CloudFront
- Use AWS Lambda for scheduled exports (serverless)

---

**Next**: Back to [[00-Index]] for documentation hub.
