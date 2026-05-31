<template>
  <CruResource
    :done-route="doneRoute"
    :resource="value"
    :mode="mode"
    :errors="errors"
    :apply-hooks="applyHooks"
    @finish="save"
    @error="(e) => errors = e"
  >
    <!-- ── Name ──────────────────────────────────────────────────────── -->
    <div class="section">
      <div class="row mb-10">
        <div class="col span-6">
          <LabeledInput
            v-model:value="resourceName"
            label="Name"
            :mode="mode === 'create' ? 'create' : 'view'"
            placeholder="e.g. my-api-key"
            required
          />
        </div>
      </div>
    </div>

    <!-- ── Source Secret ─────────────────────────────────────────────── -->
    <div class="section">
      <h3>Source Secret</h3>

      <div v-if="!isView" class="source-tabs mb-15">
        <button
          type="button"
          class="btn role-tertiary source-tab"
          :class="{ active: sourceMode === 'existing' }"
          @click="sourceMode = 'existing'"
        >
          Use Existing Secret
        </button>
        <button
          type="button"
          class="btn role-tertiary source-tab"
          :class="{ active: sourceMode === 'create' }"
          @click="sourceMode = 'create'"
        >
          Create New Secret
        </button>
      </div>

      <!-- Pick an existing secret -->
      <template v-if="sourceMode === 'existing'">
        <div class="row mb-10">
          <div class="col span-6">
            <LabeledSelect
              v-model:value="sourceNamespace"
              label="Namespace"
              :options="namespaceOptions"
              :loading="loadingMeta"
              :mode="mode"
              required
              @update:value="sourceName = ''"
            />
          </div>
          <div class="col span-6">
            <LabeledSelect
              v-model:value="sourceName"
              label="Secret"
              :options="secretOptions"
              :loading="loadingMeta"
              :disabled="!sourceNamespace"
              :mode="mode"
              required
            />
          </div>
        </div>
        <p v-if="sourceNamespace && !secretOptions.length && !loadingMeta" class="text-muted hint">
          No secrets found in <strong>{{ sourceNamespace }}</strong>. Switch to <em>Create New Secret</em> to add one.
        </p>
      </template>

      <!-- Create a new secret inline -->
      <template v-else>
        <div class="row mb-10">
          <div class="col span-6">
            <LabeledInput
              v-model:value="newSecret.name"
              label="Secret Name"
              :mode="mode"
              placeholder="e.g. my-api-key"
              required
            />
          </div>
          <div class="col span-6">
            <LabeledSelect
              v-model:value="newSecret.namespace"
              label="Namespace"
              :options="namespaceOptions"
              :loading="loadingMeta"
              :mode="mode"
              required
            />
          </div>
        </div>

        <div class="kv-header row mb-5">
          <div class="col span-5">
            <label class="text-label">Key</label>
          </div>
          <div class="col span-6">
            <label class="text-label">Value</label>
          </div>
        </div>

        <div
          v-for="(entry, idx) in newSecret.entries"
          :key="idx"
          class="row mb-5 kv-row"
        >
          <div class="col span-5">
            <LabeledInput
              v-model:value="entry.key"
              :mode="mode"
              placeholder="API_TOKEN"
            />
          </div>
          <div class="col span-6 kv-value-col">
            <LabeledInput
              v-model:value="entry.value"
              :mode="mode"
              :type="entry.showValue ? 'text' : 'password'"
              placeholder="value"
            />
            <button
              type="button"
              class="btn btn-sm role-link show-toggle"
              tabindex="-1"
              @click="entry.showValue = !entry.showValue"
            >
              {{ entry.showValue ? 'Hide' : 'Show' }}
            </button>
          </div>
          <div class="col span-1 kv-del">
            <button
              v-if="!isView && newSecret.entries.length > 1"
              type="button"
              class="btn btn-sm role-link remove-btn"
              @click="removeEntry(idx)"
            >
              ✕
            </button>
          </div>
        </div>

        <button
          v-if="!isView"
          type="button"
          class="btn btn-sm role-tertiary mt-5"
          @click="addEntry"
        >
          + Add Key
        </button>
      </template>
    </div>

    <!-- ── Targets ────────────────────────────────────────────────────── -->
    <div class="section">
      <h3>
        Targets
        <button
          v-if="!isView"
          type="button"
          class="btn btn-sm role-tertiary add-target"
          @click="addTarget"
        >
          Add Target
        </button>
      </h3>

      <div
        v-for="(target, idx) in localTargets"
        :key="idx"
        class="target-entry"
      >
        <div class="target-header">
          <strong>Target {{ idx + 1 }}</strong>
          <button
            v-if="!isView && localTargets.length > 1"
            type="button"
            class="btn btn-sm role-link remove-btn"
            @click="removeTarget(idx)"
          >
            Remove
          </button>
        </div>

        <div class="row mb-5">
          <div class="col span-4">
            <LabeledSelect
              v-model:value="target.selectorType"
              label="Target By"
              :mode="mode"
              :options="targetByOptions"
              @update:value="clearTargetSelector(target)"
            />
          </div>

          <div
            v-if="target.selectorType === 'name'"
            class="col span-8"
          >
            <LabeledSelect
              v-model:value="target.clusterName"
              label="Cluster Name"
              :options="clusterOptions"
              :loading="loadingMeta"
              :mode="mode"
              required
            />
          </div>

          <div
            v-else
            class="col span-8"
          >
            <LabeledInput
              v-model:value="target.selectorLabels"
              label="Label Selector (key=value, comma-separated)"
              :mode="mode"
              placeholder="e.g. environment=staging,region=eu"
            />
          </div>
        </div>

        <div class="row mb-5">
          <div class="col span-6">
            <LabeledInput
              v-model:value="target.namespace"
              label="Target Namespace"
              :mode="mode"
              placeholder="e.g. app"
              required
            />
          </div>
          <div class="col span-6">
            <LabeledInput
              v-model:value="target.secretName"
              label="Secret Name Override"
              :mode="mode"
              placeholder="Defaults to source secret name"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- ── Pause ─────────────────────────────────────────────────────── -->
    <div class="section">
      <h3>Sync Control</h3>
      <Checkbox
        v-model:value="paused"
        label="Pause syncing"
        :mode="mode"
      />
      <p class="hint mt-5">
        When paused, the controller stops syncing this secret. Secrets already
        synced to downstream clusters are left untouched.
      </p>
    </div>
  </CruResource>
</template>

<script>
import CruResource    from '@shell/components/CruResource';
import LabeledInput   from '@shell/rancher-components/Form/LabeledInput/LabeledInput';
import LabeledSelect  from '@shell/components/form/LabeledSelect';
import Checkbox       from '@shell/rancher-components/Form/Checkbox/Checkbox';
import CreateEditView from '@shell/mixins/create-edit-view';

export default {
  name: 'ManagedSecretEdit',

  components: { CruResource, LabeledInput, LabeledSelect, Checkbox },

  mixins: [CreateEditView],

  props: {
    value: { type: Object, required: true },
    mode:  { type: String, default: 'create' },
  },

  data() {
    return {
      errors:          [],
      resourceName:    '',
      sourceMode:      'existing',
      sourceNamespace: '',
      sourceName:      '',
      localTargets:    [],
      namespaces:      [],
      allSecrets:      [],
      loadingMeta:     false,
      managedClusters: [],
      paused:          false,

      newSecret: {
        name:      '',
        namespace: 'cattle-secrets-system',
        entries:   [{ key: '', value: '', showValue: false }],
      },

      targetByOptions: [
        { label: 'Cluster Name',   value: 'name'     },
        { label: 'Label Selector', value: 'selector' },
      ],
    };
  },

  computed: {
    isView() {
      return this.mode === 'view';
    },

    clusterOptions() {
      return this.managedClusters
        .filter(c => c.metadata?.name !== 'local')
        .map(c => c.spec?.displayName || c.metadata?.name)
        .filter(Boolean)
        .sort()
        .map(n => ({ label: n, value: n }));
    },

    namespaceOptions() {
      const ALWAYS_INCLUDE = new Set(['cattle-secrets-system', 'default']);
      const SYSTEM_PREFIXES = ['kube-', 'cattle-', 'fleet-', 'cluster-', 'user-'];

      return this.namespaces
        .filter((ns) => {
          const name = ns.metadata?.name || ns.name;

          if (!name) return false;
          if (ALWAYS_INCLUDE.has(name)) return true;
          if (ns.metadata?.annotations?.['management.cattle.io/system-namespace'] === 'true') return false;
          if (name === 'local') return false;
          if (SYSTEM_PREFIXES.some(p => name.startsWith(p))) return false;

          return true;
        })
        .map(ns => ns.metadata?.name || ns.name)
        .sort()
        .map(n => ({ label: n, value: n }));
    },

    secretOptions() {
      if (!this.sourceNamespace) return [];

      return this.allSecrets
        .filter(s => s.metadata?.namespace === this.sourceNamespace)
        .map(s => s.metadata?.name)
        .filter(Boolean)
        .sort()
        .map(n => ({ label: n, value: n }));
    },

    newSecretData() {
      const out = {};

      for (const { key, value } of this.newSecret.entries) {
        if (key) out[key] = value;
      }

      return out;
    },
  },

  async created() {
    this.initForm();
    await this.fetchMeta();
  },

  methods: {
    async fetchMeta() {
      this.loadingMeta = true;
      try {
        [this.namespaces, this.allSecrets, this.managedClusters] = await Promise.all([
          this.$store.dispatch('cluster/findAll', { type: 'namespace' }),
          this.$store.dispatch('cluster/findAll', { type: 'secret' }),
          this.$store.dispatch('management/findAll', { type: 'management.cattle.io.cluster' }),
        ]);
      } catch {
        // Graceful degradation — selects will be empty but form still works
      } finally {
        this.loadingMeta = false;
      }
    },

    // Populate local form state from value (called on create and after a failed save).
    initForm() {
      this.resourceName = this.value.metadata?.name || '';

      const spec = this.value.spec || {};
      const ref  = spec.secretRef || {};

      this.sourceNamespace = ref.namespace || '';
      this.sourceName      = ref.name      || '';
      this.paused          = !!spec.paused;

      const srcTargets = spec.targets?.length ? spec.targets : [this.blankTarget()];

      this.localTargets = srcTargets.map((t) => {
        const hasSelector = !!(t.clusterSelector?.matchLabels);

        return {
          selectorType:   t.clusterName ? 'name' : (hasSelector ? 'selector' : 'name'),
          selectorLabels: hasSelector
            ? Object.entries(t.clusterSelector.matchLabels).map(([k, v]) => `${ k }=${ v }`).join(', ')
            : '',
          clusterName:    t.clusterName || '',
          clusterSelector: t.clusterSelector || null,
          namespace:      t.namespace   || '',
          secretName:     t.secretName  || '',
        };
      });
    },

    blankTarget() {
      return {
        selectorType:    'name',
        selectorLabels:  '',
        clusterName:     '',
        clusterSelector: null,
        namespace:       '',
        secretName:      '',
      };
    },

    addTarget() {
      this.localTargets.push(this.blankTarget());
    },

    removeTarget(idx) {
      this.localTargets.splice(idx, 1);
    },

    clearTargetSelector(target) {
      target.clusterName     = '';
      target.clusterSelector = null;
      target.selectorLabels  = '';
    },

    parseSelectorLabels(raw) {
      if (!raw?.trim()) return null;

      const matchLabels = {};

      raw.trim().split(',').forEach((pair) => {
        const [k, v] = pair.trim().split('=');

        if (k) matchLabels[k.trim()] = (v || '').trim();
      });

      return Object.keys(matchLabels).length ? { matchLabels } : null;
    },

    addEntry() {
      this.newSecret.entries.push({ key: '', value: '', showValue: false });
    },

    removeEntry(idx) {
      this.newSecret.entries.splice(idx, 1);
    },

    willSave() {
      this.errors = [];

      if (this.mode === 'create' && !this.resourceName) {
        this.errors.push('Name is required.');
      }

      if (this.sourceMode === 'create') {
        if (!this.newSecret.name)      this.errors.push('New secret name is required.');
        if (!this.newSecret.namespace) this.errors.push('New secret namespace is required.');
        if (!Object.keys(this.newSecretData).length) {
          this.errors.push('Add at least one key–value entry.');
        }
      } else {
        if (!this.sourceNamespace) this.errors.push('Source namespace is required.');
        if (!this.sourceName)      this.errors.push('Source secret is required.');
      }

      for (const [i, t] of this.localTargets.entries()) {
        if (!t.namespace) {
          this.errors.push(`Target ${ i + 1 }: Namespace is required.`);
        }
        if (t.selectorType === 'name' && !t.clusterName) {
          this.errors.push(`Target ${ i + 1 }: Cluster Name is required.`);
        }
        if (t.selectorType === 'selector' && !this.parseSelectorLabels(t.selectorLabels)) {
          this.errors.push(`Target ${ i + 1 }: Label selector is required.`);
        }
      }

      return this.errors.length === 0;
    },

    // Build the clean spec from local form state, ready to write to value.
    buildSpec() {
      const targets = this.localTargets.map((t) => {
        const entry = { namespace: t.namespace };

        if (t.selectorType === 'name') {
          entry.clusterName = t.clusterName;
        } else {
          entry.clusterSelector = this.parseSelectorLabels(t.selectorLabels);
        }
        if (t.secretName) entry.secretName = t.secretName;

        return entry;
      });

      const spec = {
        secretRef: { name: this.sourceName, namespace: this.sourceNamespace },
        targets,
      };

      if (this.paused) spec.paused = true;

      return spec;
    },

    async save(buttonCb) {
      if (!this.willSave()) {
        buttonCb(false);

        return;
      }

      if (this.sourceMode === 'create') {
        try {
          const secretObj = await this.$store.dispatch('cluster/create', {
            type:       'secret',
            metadata:   { name: this.newSecret.name, namespace: this.newSecret.namespace },
            stringData: this.newSecretData,
          });

          await secretObj.save();
          this.sourceNamespace = this.newSecret.namespace;
          this.sourceName      = this.newSecret.name;
        } catch (e) {
          this.errors = [e?.data?.message || e.message || 'Failed to create secret.'];
          buttonCb(false);

          return;
        }
      }

      if (this.mode === 'create') {
        this.value.metadata.name = this.resourceName;
      }
      this.value.spec = this.buildSpec();

      try {
        await this.value.save();
        buttonCb(true);
      } catch (e) {
        this.errors = [e?.data?.message || e.message || 'Failed to save.'];
        this.initForm();
        buttonCb(false);
      }
    },
  },
};
</script>

<style scoped>
.section {
  margin-bottom: 28px;
}

.section h3 {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 14px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  opacity: 0.7;
  margin-bottom: 14px;
}

/* Source mode tab toggle */
.source-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.source-tab {
  opacity: 0.6;
  transition: opacity 0.15s;
}

.source-tab.active {
  opacity: 1;
  border-color: var(--primary);
  color: var(--primary);
}

.hint {
  font-size: 12px;
  margin-top: 4px;
  opacity: 0.7;
}

/* Key-value editor */
.kv-header .text-label {
  font-size: 12px;
  font-weight: 600;
  opacity: 0.7;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.kv-row {
  align-items: flex-end;
}

.kv-value-col {
  display: flex;
  align-items: flex-end;
  gap: 6px;
}

.kv-value-col .labeled-input {
  flex: 1;
}

.show-toggle {
  white-space: nowrap;
  padding: 0 6px;
  min-height: 34px;
  line-height: 34px;
}

.kv-del {
  display: flex;
  align-items: flex-end;
  padding-bottom: 2px;
}

/* Targets */
.add-target {
  opacity: 1;
  font-size: 12px;
}

.target-entry {
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 14px;
  margin-bottom: 12px;
  background: var(--input-bg);
}

.target-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.remove-btn {
  color: var(--error);
}
</style>
