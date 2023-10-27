# API Reference

## Packages
- [aerf.io/v1alpha1](#aerfiov1alpha1)


## aerf.io/v1alpha1

Package v1alpha1 contains the core resources of the Lambda provider.

### Resource Types
- [Lambda](#lambda)
- [ProviderConfig](#providerconfig)



#### Lambda







| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `aerf.io/v1alpha1`
| `kind` _string_ | `Lambda`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[LambdaSpec](#lambdaspec)_ |  |
| `status` _[LambdaStatus](#lambdastatus)_ |  |


#### LambdaForProvider



LambdaForProvider are the configurable fields of a Lambda.

_Appears in:_
- [LambdaSpec](#lambdaspec)

| Field | Description |
| --- | --- |
| `handler` _string_ |  |


#### LambdaSpec



A LambdaSpec defines the desired state of a MyType.

_Appears in:_
- [Lambda](#lambda)

| Field | Description |
| --- | --- |
| `writeConnectionSecretToRef` _[SecretReference](#secretreference)_ | WriteConnectionSecretToReference specifies the namespace and name of a Secret to which any connection details for this managed resource should be written. Connection details frequently include the endpoint, username, and password required to connect to the managed resource. This field is planned to be replaced in a future release in favor of PublishConnectionDetailsTo. Currently, both could be set independently and connection details would be published to both without affecting each other. |
| `publishConnectionDetailsTo` _[PublishConnectionDetailsTo](#publishconnectiondetailsto)_ | PublishConnectionDetailsTo specifies the connection secret config which contains a name, metadata and a reference to secret store config to which any connection details for this managed resource should be written. Connection details frequently include the endpoint, username, and password required to connect to the managed resource. |
| `providerConfigRef` _[Reference](#reference)_ | ProviderConfigReference specifies how the provider that will be used to create, observe, update, and delete this managed resource should be configured. |
| `managementPolicies` _[ManagementPolicies](#managementpolicies)_ | THIS IS A BETA FIELD. It is on by default but can be opted out through a Crossplane feature flag. ManagementPolicies specify the array of actions Crossplane is allowed to take on the managed and external resources. This field is planned to replace the DeletionPolicy field in a future release. Currently, both could be set independently and non-default values would be honored if the feature flag is enabled. If both are custom, the DeletionPolicy field will be ignored. See the design doc for more information: https://github.com/crossplane/crossplane/blob/499895a25d1a1a0ba1604944ef98ac7a1a71f197/design/design-doc-observe-only-resources.md?plain=1#L223 and this one: https://github.com/crossplane/crossplane/blob/444267e84783136daa93568b364a5f01228cacbe/design/one-pager-ignore-changes.md |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy specifies what will happen to the underlying external when this managed resource is deleted - either "Delete" or "Orphan" the external resource. This field is planned to be deprecated in favor of the ManagementPolicies field in a future release. Currently, both could be set independently and non-default values would be honored if the feature flag is enabled. See the design doc for more information: https://github.com/crossplane/crossplane/blob/499895a25d1a1a0ba1604944ef98ac7a1a71f197/design/design-doc-observe-only-resources.md?plain=1#L223 |
| `forProvider` _[LambdaForProvider](#lambdaforprovider)_ |  |


#### LambdaStatus



A LambdaStatus represents the observed state of a MyType.

_Appears in:_
- [Lambda](#lambda)

| Field | Description |
| --- | --- |
| `observedGeneration` _integer_ |  |
| `conditions` _[Condition](#condition) array_ | Conditions of the resource. |


#### ProviderConfig



A ProviderConfig configures a Lambda provider.



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `aerf.io/v1alpha1`
| `kind` _string_ | `ProviderConfig`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ProviderConfigSpec](#providerconfigspec)_ |  |
| `status` _[ProviderConfigStatus](#providerconfigstatus)_ |  |


#### ProviderConfigSpec



A ProviderConfigSpec defines the desired state of a ProviderConfig.

_Appears in:_
- [ProviderConfig](#providerconfig)

| Field | Description |
| --- | --- |
| `credentials` _[ProviderCredentials](#providercredentials)_ | Credentials required to authenticate to this provider. |


#### ProviderConfigStatus



A ProviderConfigStatus reflects the observed state of a ProviderConfig.

_Appears in:_
- [ProviderConfig](#providerconfig)

| Field | Description |
| --- | --- |
| `conditions` _[Condition](#condition) array_ | Conditions of the resource. |
| `users` _integer_ | Users of this provider configuration. |


#### ProviderCredentials



ProviderCredentials required to authenticate.

_Appears in:_
- [ProviderConfigSpec](#providerconfigspec)

| Field | Description |
| --- | --- |
| `source` _[CredentialsSource](#credentialssource)_ | Source of the provider credentials. |
| `fs` _[FsSelector](#fsselector)_ | Fs is a reference to a filesystem location that contains credentials that must be used to connect to the provider. |
| `env` _[EnvSelector](#envselector)_ | Env is a reference to an environment variable that contains credentials that must be used to connect to the provider. |
| `secretRef` _[SecretKeySelector](#secretkeyselector)_ | A SecretRef is a reference to a secret key that contains the credentials that must be used to connect to the provider. |


