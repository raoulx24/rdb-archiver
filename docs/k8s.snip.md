```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rdb-archiver-configmap
  namespace: test
data:
  config.yaml: |
    source:
      path: "/data"
      primaryName: "dump.rdb"
      auxNames:
      - "nodes.conf"
      watchMode: "fsnotify"          # auto | poll | fsnotify

    destination:
      root: "/backup"
      subDir: "$(HOSTNAME)"
      snapshotSubdir: "snapshots"
      retention:
        lastCount: 6
        removeUnknownFolders: true
        rules:
        - name: "hourly"
          cron: "0 * * * *"
          count: 24
        - name: "daily"
          cron: "0 0 * * *"
          count: 7
        - name: "weekly"
          cron: "0 0 * * 0"
          count: 4

    watchFS:
      fsnotify:
        debounceWindow: "200ms"
      pool:
        interval: "5s"
      stabilityWindow: "200ms"

    fs:
      maxRetries: 7
      retryBase: "50ms"
      retryDurationCap: "1s"
      compressionLevel: 2

    logging:
      level: "info"     # debug | info | warn | error
      format: "json"    # json | text

    health:
      port: 8080

    configReload:
      enabled: true
      method: "poll"    # for time being, only poll if file is mounted from configmap

---
apiVersion: apps/v1
kind: StatefulSet
spec:
  template:
    spec:
      containers:
        - name: rdb-archiver
          env:
            - name: HOSTNAME
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: metadata.name
          image: ghcr.io/raoulx24/rdb-archiver:0.0.1-35
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 8080
              name: liveness
              protocol: TCP
          readinessProbe:
            failureThreshold: 3
            httpGet:
              path: /ready
              port: liveness
              scheme: HTTP
            initialDelaySeconds: 10
            periodSeconds: 30
            successThreshold: 1
            timeoutSeconds: 3
          livenessProbe:
            failureThreshold: 3
            httpGet:
              path: /live
              port: liveness
              scheme: HTTP
            initialDelaySeconds: 30
            periodSeconds: 30
            successThreshold: 1
            timeoutSeconds: 1
          resources:
            limits:
              cpu: "3"
              memory: 512Mi
            requests:
              cpu: 100m
              memory: 128Mi
          volumeMounts:
            - mountPath: /data
              name: rds-data-persistent-volume
            - mountPath: /backup
              name: backup-volume
              subPath: testing/rdb-archiver
            - mountPath: /config
              name: rdb-archiver-config-volume
      volumes:
        - name: backup-volume
          # normally, you need a rwm volume for this,
          # as it is consumed by all sidecars from statefulset
          persistentVolumeClaim:
            claimName: my-backup-rwm-pvc
        - configMap:
            defaultMode: 420
            items:
              - key: config.yaml
                path: config.yaml
            name: rdb-archiver-configmap
          name: rdb-archiver-config-volume
  # this is here only to show where the dump.rdb is saved
  volumeClaimTemplates:
    - apiVersion: v1
      kind: PersistentVolumeClaim
      metadata:
        name: rds-data-persistent-volume
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 30Gi
        storageClassName: managed-csi-premium
        volumeMode: Filesystem
      status:
        phase: Pending

```