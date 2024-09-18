### Utility to redeploy Maven artifacts

#### Example
Redeploy from MavenCentral repository to custom Nexus repository

Create a configuration file config.yaml
```yaml
download:
  url: https://maven.google.com/
upload:
  url: https://nexus.mynexus.com/
  token: mytoken
artifacts:
  - group_id: com.google.android.material
    artifact_id: material
    version: 1.12.0
```
Run the following command
```sh
# Go installed
go run . config.yaml
# Go not installed, artifact downloaded
./redeploit config.yaml
```

#### Configuration file contents

```yaml
settings:
  save_artifacts: false # true - download artifacts to "/artifacts" directory, false - create temp directory, remove after upload
  debug_download: false # true - dont download artifacts, only log paths and destinations
  debug_upload: false # true - dont upload artifacts, only log paths and destinations
  upload_method: "PUT" # upload method, default is "POST"
download:
  url: https://maven.google.com/ # download base url
  user: my_user # base authentication user value (Base64("user:password"))
  password: my_password # base authentication password value (Base64("user:password"))
  token: my_token # Bearer token authentication
  auth_header: "PRIVATE_KEY: MyKey" # Custom authentication header
upload:
  url: https://nexus.mynexus.com/ # upload base url
  user: my_user # base authentication user value (Base64("user:password"))
  password: my_password # base authentication password value (Base64("user:password"))
  token: my_token # Bearer token authentication
  auth_header: "PRIVATE_KEY: MyKey" # Custom authentication header
artifacts:
  - group_id: com.google.android.material # Group ID
    artifact_id: material # Artifact ID
    version: 1.12.0 # Version
```
