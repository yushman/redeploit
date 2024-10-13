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
  - group_id: com.github.bumptech.glide
    artifact_id: glide
    version: 4.16.0
```
Run the following command
```sh
# Go installed
go run . config.yaml
# Go not installed, download Artifact and run (only Linux x86)
./redeploit config.yaml
```

#### Configuration file contents

```yaml
settings:
  debug_download: false # true - dont download artifacts, only log paths and destinations
  debug_upload: false # true - dont upload artifacts, only log paths and destinations
  artifacts_path: "temp" # custmon download directory, can be used to setup mavenlocal to download/upload, if not set temprorary will be created
  upload_method: "PUT" # upload method, default is "POST"
  skip_sources: false # true - skip sources artifacts from download/upload 
  skip_javadocs: false # true - skip javadoc artifacts from download/upload 
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

#### Some specific cases

**Only upload artifacts from mavenlocal or any other maven structured directory**
```yaml
settings:
  debug_download: true # true - to skip download phase
  artifacts_path: "home/user/.m2" # path to maven-local
download:
  url: https://anywhere.com/
upload:
  url: https://nexus.mynexus.com/
  token: mytoken
artifacts:
  - group_id: com.google.android.material
    artifact_id: material
    version: 1.12.0
```
**Only download artifacts to mavenlocal or any other maven structured directory**
```yaml
settings:
  debug_upload: true # true - to skip upload phase
  artifacts_path: "home/user/.m2" # path to maven-local
download:
  url: https://maven.google.com/
upload:
  url: https://anywhere.com/
artifacts:
  - group_id: com.google.android.material
    artifact_id: material
    version: 1.12.0
```
