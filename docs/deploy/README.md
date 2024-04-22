To deploy OSS:
```shell
export DET_VARIANT=OSS
make publish
```

To deploy EE:
```shell
export DET_VARIANT=OSS
make publish
```

Before running local terraform commands, set the DET_VARIANT variable to OSS or
EE and run `make init`.  Terraform commands will then work as normal once the
backend is changed to reflect the variant.