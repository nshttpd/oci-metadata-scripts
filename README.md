### OCI Metadata Script Handler

#### Overview

The intent of this is to make it easy to be able to run a startup or shutdown script on an instance for purposes such 
as configuration, ping back announcements or anything else one may want to do during these lifecycle steps.

Scripts are defined in the instance metadata as being either of `startup` or `shutdown` and can be sourced from 
either local storage, metadata value, remote url or
[Object Storage](https://docs.cloud.oracle.com/iaas/Content/Object/Concepts/objectstorageoverview.htm). If more
than one source is supplied both will be executed with remote being run first.

`startup-script` Examples :

```
"startup-script" : "/opt/service/start/bootstrap.py"
```

```
"startup-script" : "IyEvdXNyL2Jpbi9lbnYgYmFzaAoKZWNobyAibXkgYXdlc29tZSBzdGFydHVwIHNjcmlwdCIK"
```

The first will execute the script that is contained locally on the host instance. It will be copied into a
temporary location before execution.

The second is the actual script data base64 encoded and supplied in the instance metadata. It will be base64 decoded
into a temporary location under a temporary filename and then executed.

`startup-script-url` Examples:

```
"startup-script-url" : "oci://bucket@namespace/bootstrap.py"
```

```
"startup-script-url" : "https://trusted-remote-server.io/boot/bootstrap.py"
```

The first will fetch the script from an Object Storage bucket in the namespace, store in a temporary working
directory and then execute it. It is the responsibility of the owner to make sure that the instances have access to
the bucket.

The second will make a request to the specified location, download to a temporary directory and execute the
payload. Be warned that this should be a trusted known site and that egress to the remote location is allowed.

#### Object Storage Example

If Object storage will be the source for the scripts a
[policy](https://docs.cloud.oracle.com/iaas/Content/Identity/Concepts/policygetstarted.htm)
will need to be applied such that the instances
can easily have read-only access to the buckets in the namespace to retrieve the scripts from. A definition for
[Dynamic Groups](https://docs.cloud.oracle.com/iaas/Content/Identity/Tasks/managingdynamicgroups.htm) is also helpful
for being able to allow instances in this group to have the policy for read-only access to the buckets. Using
[Terraform](https://www.terraform.io/) it could be something like :


```
resource "oci_identity_dynamic_group" "ro_script_buckets" {
  compartment_id = "${var.tenancy_ocid}"
  name           = "ro-script-buckets"
  description    = "dynamic group for reading startup/shutdown script buckets"
  matching_rule  = <<EOF
Any {instance.compartment.id = 'ocid1.compartment.oc1..aaaaaxutgua',
instance.compartment.id = 'ocid1.compartment.oc1..aaaaallwh6mv7avuozfsus4yaia',
instance.compartment.id = 'ocid1.compartment.oc1..aaaaaaaaszz2v4gyfza'}
EOF
}
```

Where the `instance.compartment.id` values are those of compartments where instances will be created in and will
require access to the startup or shutdown scripts buckets.

```
resource "oci_identity_policy" "script_buckets_read" {
    depends_on = ["oci_identity_dynamic_group.ro_script_buckets"]
    compartment_id = "${var.tenancy_ocid}"
    name = "script-buckets-read"
    description = "read only policy for scripts buckets"
    statements = [
        "allow dynamic-group ro-script-buckets to read buckets in compartment Bucket_Compartment where any {target.bucket.name='startup-scripts', target.bucket.name='shutdown-scripts'}",
        "allow dynamic-group ro-script-buckets to read objects in compartment Bucket_Compartment where any {target.bucket.name='startup-scripts', target.bucket.name='shutdown-scripts'}"
    ]
}
```

This will allow the instances in the dynamic group created above to access the buckets with the names `startup-scripts`
and `shutdown-scripts`.
