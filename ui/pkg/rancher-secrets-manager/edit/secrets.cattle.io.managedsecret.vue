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
    <div class="section">
      <h3>Source Secret</h3>
      <div class="row mb-10">
        <div class="col span-6">
          <LabeledInput
            v-model="value.spec.secretRef.namespace"
            label="Source Namespace"
            :mode="mode"
            placeholder="e.g. cattle-secrets-system"
            required
          />
        </div>
        <div class="col span-6">
          <LabeledInput
            v-model="value.spec.secretRef.name"
            label="Source Secret Name"
            :mode="mode"
            placeholder="e.g. my-database-password"
            required
          />
        </div>
      </div>
    </div>

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
      errors: [],

      targetByOptions: [
        { label: 'Cluster Name', value: 'name' },
        { label: 'Label Selector', value: 'selector' },
      ],
    };
  },

  computed: {
    isView() {
      return this.mode === 'view';
    },
  },

  created() {
    this.initResource();
  },

  methods: {
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
        _selectorType:  'name',
        _selectorLabels: '',
        clusterName:    '',
        clusterSelector: null,
        namespace:      '',
        secretName:     '',
      };
    },

    addTarget() {
      this.value.spec.targets.push(this.newTarget());
    },

    removeTarget(idx) {
      this.value.spec.targets.splice(idx, 1);
    },

    clearTargetSelector(target) {
      target.clusterName      = '';
      target.clusterSelector  = null;
      target._selectorLabels  = '';
    },

    parseSelectorLabels(target) {
      const raw = (target._selectorLabels || '').trim();

      if (!raw) {
        target.clusterSelector = null;

        return;
      }

      const matchLabels = {};

      raw.split(',').forEach((pair) => {
        const [k, v] = pair.trim().split('=');

        if (k) matchLabels[k.trim()] = (v || '').trim();
      });

      target.clusterSelector = { matchLabels };
    },

    willSave() {
      this.errors = [];

      const spec = this.value.spec || {};

      if (!spec.secretRef?.name || !spec.secretRef?.namespace) {
        this.errors.push('Source Secret name and namespace are required.');
      }

      for (const [i, t] of (spec.targets || []).entries()) {
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

        delete t._selectorType;
        delete t._selectorLabels;

        if (!t.secretName) delete t.secretName;
        if (!t.clusterName) delete t.clusterName;
        if (!t.clusterSelector) delete t.clusterSelector;
      }

      return this.errors.length === 0;
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
