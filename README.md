# paramedic-agent

Agent for [Paramedic](https://github.com/ryotarai/paramedic)

## SSM Document to install

```
{
    "schemaVersion":"2.2",
    "description":"Deploy paramedic-agent",
    "parameters":{
        "version":{
            "type":"String",
            "description":"(Required) version"
        },
        "sha256sum":{
            "type":"String",
            "description":"(Required) sha256 checksum"
        }
    },
       "mainSteps":[
      {
         "action":"aws:runShellScript",
         "name":"script",
         "inputs":{
            "runCommand":["FILE='paramedic-agent_linux_amd64_{{ version }}' && curl -L -O https://github.com/ryotarai/paramedic-agent/releases/download/v{{ version }}/${FILE}.gz && echo {{ sha256sum }} ${FILE}.gz | sha256sum --check --status - && gunzip ${FILE}.gz && chmod 755 ${FILE} && mv ${FILE} /usr/local/sbin/paramedic-agent"],
            "workingDirectory":"/tmp"
         }
      }
   ]
}
```

## Configuration

Put a config file at `/etc/paramedic-agent/config.yaml` if you want

```yaml
# (Optional) Specify which credential provider aws-sdk uses
AWSCredentialProvider: 'EC2Role'
```
