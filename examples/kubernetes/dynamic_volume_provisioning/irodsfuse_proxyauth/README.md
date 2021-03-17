After creating a Kubernetes secret object, you should restart the CSI driver to apply.
This is because the secret object you created is global and loaded during the driver initialization.

