const PRODUCT_NAME = 'secrets-manager';
const RESOURCE     = 'secrets.cattle.io.managedsecret';

export function init($plugin: any, store: any) {
  const { product, configureType, basicType } = $plugin.DSL(store, PRODUCT_NAME);

  product({
    icon:         'lock',
    inStore:      'cluster',
    weight:       80,
    ifHaveGroup:  'secrets.cattle.io',
    to: {
      name:   `${ PRODUCT_NAME }-c-cluster-resource`,
      params: { product: PRODUCT_NAME, cluster: 'local', resource: RESOURCE },
    },
  });

  configureType(RESOURCE, {
    displayName: 'Managed Secrets',
    isCreatable: true,
    isEditable:  true,
    isRemovable: true,
    showAge:     true,
    showState:   true,
    canYaml:     true,
  });

  basicType([RESOURCE]);
}
