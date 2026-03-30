# Kubernetes для Go сервисов

## Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: order-service
  labels:
    app: order-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: order-service
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0  # zero downtime
  template:
    metadata:
      labels:
        app: order-service
    spec:
      terminationGracePeriodSeconds: 60
      containers:
        - name: order-service
          image: myregistry/order-service:v1.2.3
          ports:
            - containerPort: 8080
              name: http
            - containerPort: 50051
              name: grpc
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: db-credentials
                  key: url
            - name: GOMAXPROCS
              valueFrom:
                resourceFieldRef:
                  resource: limits.cpu  # автоматически по CPU limit
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi
          livenessProbe:
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /readyz
              port: http
            initialDelaySeconds: 3
            periodSeconds: 5
            failureThreshold: 1
          startupProbe:
            httpGet:
              path: /healthz
              port: http
            failureThreshold: 30
            periodSeconds: 2  # 30*2=60s на старт
          lifecycle:
            preStop:
              exec:
                command: ["sh", "-c", "sleep 5"]
```

## Resource Limits

```
CPU:
  requests: гарантированный минимум (для scheduling)
  limits: максимум (throttling при превышении)

  Go рекомендация:
    requests = средняя нагрузка
    limits = 2-3x от requests (или без limits для CPU)
    GOMAXPROCS = limits.cpu (automaxprocs library)

Memory:
  requests: гарантированный минимум
  limits: максимум (OOMKill при превышении!)

  Go рекомендация:
    limits = GOMEMLIMIT * 1.1 (10% запас)
    Или: GOMEMLIMIT = limits * 0.9

  GOMEMLIMIT в env:
    env:
      - name: GOMEMLIMIT
        value: "460MiB"  # для limits: 512Mi
```

```go
// automaxprocs — автоматически устанавливает GOMAXPROCS по CPU limits
import _ "go.uber.org/automaxprocs"

func main() {
    // GOMAXPROCS автоматически = CPU limit (не все ядра хоста!)
}
```

## Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: order-service
spec:
  selector:
    app: order-service
  ports:
    - name: http
      port: 80
      targetPort: 8080
    - name: grpc
      port: 50051
      targetPort: 50051
      appProtocol: grpc  # для L7 LB (Istio)
```

## HPA (Horizontal Pod Autoscaler)

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: order-service
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: order-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

## ConfigMap и Secrets

```yaml
# Config
apiVersion: v1
kind: ConfigMap
metadata:
  name: order-service-config
data:
  config.yaml: |
    server:
      http_port: 8080
      grpc_port: 50051
    log:
      level: info

# Монтирование
volumes:
  - name: config
    configMap:
      name: order-service-config
containers:
  - volumeMounts:
      - name: config
        mountPath: /etc/config
        readOnly: true
```

## Go-специфичные советы для K8s

```
1. automaxprocs — GOMAXPROCS по CPU limits
2. GOMEMLIMIT — по memory limits (× 0.9)
3. Graceful shutdown — terminationGracePeriodSeconds > shutdown timeout
4. preStop hook — sleep 3-5s для drain
5. readinessProbe — отключаться перед shutdown
6. Stateless — состояние в DB/Redis, не в памяти pod
7. Health checks — /healthz (liveness) + /readyz (readiness)
8. Structured logging (JSON) — для log aggregation
```
