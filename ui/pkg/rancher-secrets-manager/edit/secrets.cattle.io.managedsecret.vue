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
              v-model="value.spec.secretRef.namespace"
              label="Namespace"
              :options="namespaceOptions"
              :loading="loadingMeta"
              :mode="mode"
              required
              @update:model-value="value.spec.secretRef.name = ''"
            />
          </div>
          <div class="col span-6">
            <LabeledSelect
              v-model="value.spec.secretRef.name"
              label="Secret"
              :options="secretOptions"
              :loading="loadingMeta"
              :disabled="!value.spec.secretRef.namespace"
              :mode="mode"
              required
            />
          </div>
        </div>
        <p v-if="value.spec.secretRef.namespace && !secretOptions.length && !loadingMeta" class="text-muted hint">
          No secrets found in <strong>{{ value.spec.secretRef.namespace }}</strong>. Switch to <em>Create New Secret</em> to add one.
        </p>
      </template>

      <!-- Create a new secret inline -->
      <template v-else>
        <div class="row mb-10">
          <div class="col span-6">
            <LabeledInput
              v-model="newSecret.name"
              label="Secret Name"
              :mode="mode"
              placeholder="e.g. my-api-key"
              required
            />
          </div>
          <div class="col span-6">
            <LabeledSelect
              v-model="newSecret.namespace"
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
              v-model="entry.key"
              :mode="mode"
              placeholder="API_TOKEN"
            />
          </div>
          <div class="col span-6 kv-value-col">
            <LabeledInput
              v-model="entry.value"
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
        v-for="(target, idx) in value.spec.targets"
        :key="idx"
        class="target-entry"
      >
        <div class="target-header">
          <strong>Target {{ idx + 1 }}</strong>
          <button
            v-if="!isView && value.spec.targets.length > 1"
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
              v-model="target._selectorType"
              label="Target By"
              :mode="mode"
              :options="targetByOptions"
              @update:model-value="clearTargetSelector(target)"
            />
          </div>

          <div
            v-if="target._selectorType === 'name'"
            class="col span-8"
          >
            <LabeledInput
              v-model="target.clusterName"
              label="Cluster Name"
              :mode="mode"
              placeholder="e.g. production-eu"
              required
            />
          </div>

          <div
            v-else
            class="col span-8"
          >
            <LabeledInput
              v-model="target._selectorLabels"
              label="Label Selector (key=value, comma-separated)"
              :mode="mode"
              placeholder="e.g. environment=staging,region=eu"
              @blur="parseSelectorLabels(target)"
            />
          </div>
        </div>

        <div class="row mb-5">
          <div class="col span-6">
            <LabeledInput
              v-model="target.namespace"
              label="Target Namespace"
              :mode="mode"
              placeholder="e.g. app"
              required
            />
          </div>
          <div class="col span-6">
            <LabeledInput
              v-model="target.secretName"
              label="Secret Name Override"
              :mode="mode"
              placeholder="Defaults to source secret name"
            />
          </div>
        </div>
      </div>
    </div>
  </CruResource>
</template>

<script>
import CruResource    from '@shell/components/CruResource';
import LabeledInput   from '@shell/rancher-components/Form/LabeledInput/LabeledInput';
import LabeledSelect  from '@shell/components/form/LabeledSelect';
import CreateEditView from '@shell/mixins/create-edit-view';

export default {
  name: 'ManagedSecretEdit',

  components: { CruResource, LabeledInput, LabeledSelect },

  mixins: [CreateEditView],

  props: {
    value: { type: Object, required: true },
    mode:  { type: String, default: 'create' },
  },

  data() {
    return {
      errors:      [],
      sourceMode:  'existing',
      namespaces:  [],
      allSecrets:  [],
      loadingMeta: false,

      newSecret: {
        name:      '',
        namespace: 'cattle-secrets-system',
        entries:   [{ key: '', value: '', showValue: false }],
      },

      sourceModeOptions: [
        { label: 'Use existing secret', value: 'existing' },
        { label: 'Create new secret',   value: 'create'   },
      ],

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

    namespaceOptions() {
      return this.namespaces
        .map(ns => ns.metadata?.name || ns.name)
        .filter(Boolean)
        .sort()
        .map(n => ({ label: n, value: n }));
    },

    secretOptions() {
      const ns = this.value.spec?.secretRef?.namespace;

      if (!ns) return [];

      return this.allSecrets
        .filter(s => s.metadata?.namespace === ns)
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
    this.initResource();
    await this.fetchMeta();
  },

  methods: {
    async fetchMeta() {
      this.loadingMeta = true;
      try {
        [this.namespaces, this.allSecrets] = await Promise.all([
          this.$store.dispatch('cluster/findAll', { type: 'namespace' }),
          this.$store.dispatch('cluster/findAll', { type: 'secret' }),
        ]);
      } catch {
        // Graceful degradation — selects will be empty but form still works
      } finally {
        this.loadingMeta = false;
      }
    },

    initResource() {
      if (!this.value.spec) {
        this.value.spec = { secretRef: { name: '', namespace: '' }, targets: [] };
      }
      if (!this.value.spec.secretRef) {
        this.value.spec.secretRef = { name: '', namespace: '' };
      }
      if (!this.value.spec.targets?.length) {
        this.value.spec.targets = [this.newTarget()];
      }

      this.value.spec.targets.forEach((t) => {
        if (!t._selectorType) {
          t._selectorType = t.clusterName ? 'name' : 'selector';
        }
        if (!t._selectorLabels && t.clusterSelector?.matchLabels) {
          t._selectorLabels = Object.entries(t.clusterSelector.matchLabels)
            .map(([k, v]) => `${ k }=${ v }`)
            .join(', ');
        }
      });
    },

    newTarget() {
      return {
        _selectorType:   'name',
        _selectorLabels: '',
        clusterName:     '',
        clusterSelector: null,
        namespace:       '',
        secretName:      '',
      };
    },

    addTarget() {
      this.value.spec.targets.push(this.newTarget());
    },

    removeTarget(idx) {
      this.value.spec.targets.splice(idx, 1);
    },

    clearTargetSelector(target) {
      target.clusterName     = '';
      target.clusterSelector = null;
      target._selectorLabels = '';
    },

    parseSelectorLabels(target) {
      const raw = (target._selectorLabels || '').trim();

      if (!raw) { target.clusterSelector = null; return; }

      const matchLabels = {};

      raw.split(',').forEach((pair) => {
        const [k, v] = pair.trim().split('=');

        if (k) matchLabels[k.trim()] = (v || '').trim();
      });
      target.clusterSelector = { matchLabels };
    },

    addEntry() {
      this.newSecret.entries.push({ key: '', value: '', showValue: false });
    },

    removeEntry(idx) {
      this.newSecret.entries.splice(idx, 1);
    },

    willSave() {
      this.errors = [];

      if (this.sourceMode === 'create') {
        if (!this.newSecret.name)      this.errors.push('New secret name is required.');
        if (!this.newSecret.namespace) this.errors.push('New secret namespace is required.');
        if (!Object.keys(this.newSecretData).length) {
          this.errors.push('Add at least one key–value entry.');
        }
      } else {
        if (!this.value.spec?.secretRef?.namespace) this.errors.push('Source namespace is required.');
        if (!this.value.spec?.secretRef?.name)      this.errors.push('Source secret is required.');
      }

      for (const [i, t] of (this.value.spec?.targets || []).entries()) {
        if (!t.namespace) {
          this.errors.push(`Target ${ i + 1 }: Namespace is required.`);
        }
        if (t._selectorType === 'name' && !t.clusterName) {
          this.errors.push(`Target ${ i + 1 }: Cluster Name is required.`);
        }
        if (t._selectorType === 'selector') {
          this.parseSelectorLabels(t);
          if (!t.clusterSelector) {
            this.errors.push(`Target ${ i + 1 }: Label selector is required.`);
          }
        }
      }

      return this.errors.length === 0;
    },

    async save(buttonCb) {
      if (!this.willSave()) {
        buttonCb(false);

        return;
      }

      // Strip UI-only fields from targets before sending to API
      for (const t of this.value.spec.targets) {
        delete t._selectorType;
        delete t._selectorLabels;
        if (!t.secretName)      delete t.secretName;
        if (!t.clusterName)     delete t.clusterName;
        if (!t.clusterSelector) delete t.clusterSelector;
      }

      // If the user chose to create a new secret, do that first
      if (this.sourceMode === 'create') {
        try {
          const secretObj = await this.$store.dispatch('cluster/create', {
            type:       'secret',
            metadata:   { name: this.newSecret.name, namespace: this.newSecret.namespace },
            stringData: this.newSecretData,
          });

          await secretObj.save();
          this.value.spec.secretRef = { name: this.newSecret.name, namespace: this.newSecret.namespace };
        } catch (e) {
          this.errors = [e?.data?.message || e.message || 'Failed to create secret.'];
          this.initResource();
          buttonCb(false);

          return;
        }
      }

      // Save the ManagedSecret itself
      try {
        await this.value.save();
        buttonCb(true);
      } catch (e) {
        this.errors = [e?.data?.message || e.message || 'Failed to save.'];
        this.initResource();
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
