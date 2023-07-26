# Cloud Build Discord Notifier

- 此專案為使用官方範例改寫的 Go 專案，負責將 Cloud build 狀態傳送至通訊軟體

## Demo

![building, success](https://imgur.com/pijQZfK.png)

![failure](https://imgur.com/Z37TdIY.png)

> 訊息標題皆可點擊跳轉至 GCP Cloud Build 或 GKE 頁面

## 資源概覽

### IAM

Service Account

- `cloud-build-notifications@<PROJECT_ID>.iam.gserviceaccount.com`

### Pub / Sub and Topics

Topic <-> Sub

- cloud-builds <-> send-cloud-build-to-notifiers

> <備註> Topic 建立名稱只需設定 `cloud-builds`，不需要其他設定即可接收

### Cloud Run

- Cloud Run: cloud-build-discord-notifier

### Cloud Storage

- Bucket: cloud_latitude_files
  - Directory: `cloud-build-notifications/discord.yaml`

> <備註> 存放 filter 設定檔，程式會從這檔案來判斷 cloud build 狀態

參考：

- <https://cloud.google.com/build/docs/configuring-notifications/configure-slack#using_cel_to_filter_build_events>

## 架構說明

![Cloud Build notifiers](https://cloud.google.com/build/images/cloud-build-notifiers.svg)

1. Cloud Build 會傳送狀態訊息出去，通知管道為 Pub / Sub
2. Pub / Sub 會收到 Cloud Build status，並將訊息推送至 Cloud Run
3. Cloud Run 會從 GCS 指定的 Bucket 取得 filter 設定檔過濾 Cloud Build status
4. Cloud Run 也會取得在 Secret Manager 設定的 Discord Webhook Url
5. 最後，接收狀態訊息處理後，並轉發到 Discord Webhook

### 程式說明

- 程式參考官方及別人寫好的程式直接修改，並引用官方範例寫好的 package
- 程式只定義新的 struct，及改寫 `buildMessage` 的 function
- 從 `build.Substitutions` 中取得 Repository, Refer, Trigger Name 資訊
- 由服務及 Trigger Name 來判斷傳送至 Discord 的通知內容

參考：

- <https://github.com/GoogleCloudPlatform/cloud-build-notifiers>
- <https://github.com/samccone/cloud-build-discord-notifier>

#### <注意>

1. 範例的 go package `google.golang.org/genproto/googleapis/devtools/cloudbuild/v1` 已棄用
2. package 須改使用 `cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb`，程式大致上都相容

參考：

- <https://github.com/googleapis/google-cloud-go/blob/main/migration.md>

## 資料夾結構

- `main.go`：主程式邏輯皆在這，僅修改 `buildMessage`
- `/data/data.go`：local package 提供給 main.go 使用，這裡做 url 的判斷回傳，以及從 Github 取得 commit author 資訊
- `Dockerfile`：將此專案打包成 Docker Image
- `cloudbuild.yaml`：CI / CD 自動化部署到 Cloud Run
- `go.mod`：Go 會使用到的套件版本
- `discord.yaml`：上傳至 GCS Bucket 的檔案

```bash
├── README.md
├── cloudbuild.yaml
├── data
│   └── data.go
├── discord.yaml
├── dockerfile
├── go.mod
├── go.sum
└── main.go
```

## 設定說明

### Discord Webhook

1. 編輯頻道 > 整合 > 查看 webhook > 新增 webhook
2. 取得 webhook url

詳細步驟可參考：

- <https://10mohi6.medium.com/super-easy-python-discord-notifications-api-and-webhook-9c2d85ffced9>

><注意> Discord Webhook 訊息限制為每分鐘 30 則訊息

### Github personal access token

1. Settings >  Developer settings > Personal access tokens (classic)
2. 權限：repo (All)、admin:org (read:org)

取得 token 後，目前暫時寫死在 `/data/data.go`

```go
ctx := context.Background()
ts := oauth2.StaticTokenSource(
  &oauth2.Token{AccessToken: "<YOUR_GITHUB_TOKEN>"},
)

tc := oauth2.NewClient(ctx, ts)
client := github.NewClient(tc)
```

參考：

- <https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token>

### Cloud Run IAM

1. 建立 Service Account
2. 向 Service Account 授予 `Storage Object Viewer` 的權限
3. 向 Service Account 授予 `Secret Manager Secret Accessor` 的權限

```bash
$ gcloud iam service-accounts create cloud-build-notifications \
    --display-name "Cloud Build Notifications Identity"

$ gcloud projects add-iam-policy-binding <PROJECT_ID> \
    --member "serviceAccount:cloud-build-notifications@<PROJECT_ID>.iam.gserviceaccount.com" \
    --role "roles/storage.objectViewer"

$ gcloud projects add-iam-policy-binding <PROJECT_ID> \
    --member "serviceAccount:cloud-build-notifications@<PROJECT_ID>.iam.gserviceaccount.com" \
    --role "roles/secretmanager.secretAccessor"
```

### First Cloud Run Deploy

```bash
# deploy to cloud run
$ gcloud beta run deploy cloud-build-discord-notifier \
--image=us-docker.pkg.dev/<PROJECT_ID>/<PROJECT_NAME>/cloud-build-discord-notifier:latest \
--region=asia-southeast1 \
--project=<PROJECT_ID> \
--execution-environment gen2 \
--service-account cloud-build-notifications \
--min-instances=0 \
--max-instances=2 \
--no-allow-unauthenticated \
--update-env-vars=CONFIG_PATH=gs://<BUCKET_NAME>/cloud-build-notifications/discord.yaml
```

#### <備註>

1. `CONFIG_PATH` 環境變數一定要設定！！
2. 執行順序：如果您在容器中設置預設環境變數，並在 Cloud Run 服務上設置具有相同名稱的環境變數，則該服務中設置的值優先
3. 如要使用第二代 Cloud Run 環境，部署指令需要加上 `beta`

參考：

- <https://cloud.google.com/build/docs/configuring-notifications/configure-slack>

### Pub / Sub IAM

- 建立 Service Account，並給予 Token Create 權限

  ```bash
  $ gcloud iam service-accounts create cloud-run-pubsub-invoker \
      --display-name "Cloud Run Pub/Sub Invoker"

  $ gcloud projects add-iam-policy-binding <PROJECT_ID> \
      --member=serviceAccount:service-000000000000@gcp-sa-pubsub.iam.gserviceaccount.com \
      --role=roles/iam.serviceAccountTokenCreator
  ```

- 設定 Cloud Run 需要 Pub/Sub Service Account 才能訪問，此步驟沒做**無法**使用
  
  ```bash
  $ gcloud run services add-iam-policy-binding cloud-build-discord-notifier \
      --member=serviceAccount:cloud-run-pubsub-invoker@<PROJECT_ID>.iam.gserviceaccount.com \
      --role=roles/run.invoker \
      --region=asia-southeast1
  ```

參考：

- <https://cloud.google.com/run/docs/tutorials/pubsub#integrating-pubsub>
- <https://cloud.google.com/build/docs/configuring-notifications/configure-slack>

### Pub / Sub

1. 建立 Pub / Sub 主題 `cloud-builds`
2. 設定 Pub / Sub 主題的 "訂閱項目"，並設定權限

```bash
$ gcloud pubsub topics create cloud-builds

$ gcloud pubsub subscriptions create send-cloud-build-to-notifiers \
   --topic=cloud-builds \
   --ack-deadline=180 \
   --push-endpoint=$CLOUD_RUN_URL \
   --push-auth-service-account=cloud-run-pubsub-invoker@<PROJECT_ID>.iam.gserviceaccount.com
```

參考：

- <https://cloud.google.com/run/docs/tutorials/pubsub#integrating-pubsub>
- <https://cloud.google.com/build/docs/configuring-notifications/configure-slack>

## Cloud Build

當此專案分支打上 tag 會觸發 Cloud Build 執行部署服務至 Cloud Run

```yaml
steps:
  # Build images
  - name: gcr.io/cloud-builders/docker
    args: [ 'build', '-t', 'us-docker.pkg.dev/$PROJECT_ID/$PROJECT_NAME/cloud-build-discord-notifier:${TAG_NAME}', '.' ]

  # Deploy container image to Cloud Run
  - name: 'gcr.io/google.com/cloudsdktool/cloud-sdk:alpine'
    entrypoint: gcloud
    args:
    - 'run'
    - 'deploy'
    - 'cloud-build-discord-notifier'
    - '--image'
    - 'us-docker.pkg.dev/$PROJECT_ID/$PROJECT_NAME/cloud-build-discord-notifier:${TAG_NAME}'
    - '--region'
    - 'asia-southeast1'

images:
  - 'us-docker.pkg.dev/$PROJECT_ID/$PROJECT_NAME/cloud-build-discord-notifier:${TAG_NAME}'

options:
  logging: GCS_ONLY
  # pool: # use private pool to connect private IP services
  #   name: ''

```

參考：

- <https://cloud.google.com/build/docs/deploying-builds/deploy-cloud-run>
