const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const root = path.resolve(__dirname, '..');

// Load the options and installed SDK model without booting Rancher's dashboard.
function loadDefault(source, bindings) {
  const module = { exports: {} };

  vm.runInNewContext(source
    .replace(/^import .*;$/gm, '')
    .replace(/^export const /gm, 'const ')
    .replace('export default ', 'module.exports = '), { module, ...bindings });

  return module.exports;
}

const component = loadDefault(fs.readFileSync(path.join(root,
  'pkg/rancher-secrets-manager/edit/secrets.cattle.io.managedsecret.vue'), 'utf8')
  .split('<script>')[1].split('</script>')[0], {
  CruResource: {}, LabeledInput: {}, LabeledSelect: {}, Checkbox: {}, FileSelector: {}, CreateEditView: {},
});

const Secret = loadDefault(fs.readFileSync(path.join(root,
  'node_modules/@rancher/shell/models/secret.js'), 'utf8'), {
  SteveModel: class {},
  base64Encode: value => Buffer.from(value).toString('base64'),
  set(object, key, value) {
    object[key.startsWith('"') ? JSON.parse(key) : key] = value;
  },
});

function form(sourceMode = 'create') {
  const context = {
    ...component.data(),
    mode: 'create',
    sourceMode,
    resourceName: 'test-create',
    sourceName: 'existing-source',
    localTargets: [{ selectorType: 'name', clusterName: 'downstream', namespace: 'default' }],
    done() {},
  };

  for (const [name, method] of Object.entries(component.methods)) {
    context[name] = method.bind(context);
  }
  Object.defineProperty(context, 'newSecretData', {
    get: () => component.computed.newSecretData.call(context),
  });
  context.newSecret.name = 'test-create';
  context.newSecret.entries = [
    { key: 'token', value: 'example-token' },
    { key: 'config.txt', value: 'first line\n日本語 🔒\n' },
    { key: 'empty', value: '' },
  ];

  return context;
}

test('inline creation saves encoded Opaque data before the ManagedSecret', async () => {
  const context = form();
  const calls = [];

  context.$store = { async dispatch(action, payload) {
    assert.equal(action, 'cluster/create');
    assert.equal(payload.type, 'secret');
    assert.equal(payload._type, 'Opaque');
    assert.equal(payload.stringData, undefined);
    const secret = Object.assign(new Secret(), payload);

    secret.save = async () => {
      for (const entry of context.newSecret.entries) {
        assert.equal(Buffer.from(secret.data[entry.key], 'base64').toString(), entry.value);
      }
      assert.equal(Object.keys(secret.data).length, 3);
      calls.push('secret');
    };

    return secret;
  } };
  context.value = { metadata: {}, async save() {
    assert.equal(this.spec.secretRef.name, 'test-create');
    assert.equal(this.spec.secretRef.namespace, 'cattle-secrets-system');
    calls.push('managedsecret');
  } };

  await context.save(result => assert.equal(result, true));
  assert.deepEqual(calls, ['secret', 'managedsecret']);
});

test('existing-source creation does not create or modify a Secret', async () => {
  const context = form('existing');

  context.$store = { dispatch() { assert.fail('unexpected Secret creation'); } };
  context.value = { metadata: {}, async save() {
    assert.equal(this.spec.secretRef.name, 'existing-source');
  } };

  await context.save(result => assert.equal(result, true));
});

test('failed source save prevents ManagedSecret creation', async () => {
  const context = form();

  context.$store = { async dispatch() {
    return { setData() {}, async save() { throw new Error('Source save failed'); } };
  } };
  context.value = { async save() { assert.fail('unexpected ManagedSecret creation'); } };

  await context.save(result => assert.equal(result, false));
  assert.equal(context.errors[0], 'Source save failed');
});
