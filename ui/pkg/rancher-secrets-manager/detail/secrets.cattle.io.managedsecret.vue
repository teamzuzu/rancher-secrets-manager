<template>
  <div>
    <ResourceTabs
      v-model="activeTab"
      class="mt-15"
      :side-tabs="true"
      :default-tab="'overview'"
    >
      <Tab
        name="overview"
        label="Overview"
        :weight="99"
      >
        <div class="overview">
          <div class="detail-section">
            <h3>Source Secret</h3>
            <div class="detail-row">
              <span class="detail-label">Name</span>
              <span>{{ value.spec.secretRef.name }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Namespace</span>
              <span>{{ value.spec.secretRef.namespace }}</span>
            </div>
          </div>

          <div class="detail-section">
            <h3>Sync Summary</h3>
            <div class="detail-row">
              <span class="detail-label">Total Targets</span>
              <span>{{ value.status.targetCount || 0 }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Synced</span>
              <span>{{ value.status.syncedCount || 0 }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Overall Status</span>
              <span :class="['badge', syncBadgeClass]">{{ value.syncState }}</span>
            </div>
          </div>

          <div class="detail-section">
            <h3>Targets ({{ value.spec.targets.length }})</h3>
            <table class="targets-table">
              <thead>
                <tr>
                  <th>Cluster</th>
                  <th>Namespace</th>
                  <th>Secret Name Override</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(target, idx) in value.spec.targets"
                  :key="idx"
                >
                  <td>
                    {{ target.clusterName || selectorDisplay(target.clusterSelector) }}
                  </td>
                  <td>{{ target.namespace }}</td>
                  <td>{{ target.secretName || '—' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </Tab>

      <Tab
        name="syncStatus"
        label="Sync Status"
        :weight="98"
      >
        <div v-if="!syncEntries.length" class="no-entries">
          No sync status yet — the controller may still be processing.
        </div>
        <table
          v-else
          class="sync-table"
        >
          <thead>
            <tr>
              <th>Cluster</th>
              <th>Namespace</th>
              <th>Secret Name</th>
              <th>Status</th>
              <th>Message</th>
              <th>Last Synced</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="entry in syncEntries"
              :key="`${ entry.clusterName }/${ entry.namespace }`"
            >
              <td>{{ entry.clusterName }}</td>
              <td>{{ entry.namespace }}</td>
              <td>{{ entry.secretName }}</td>
              <td>
                <span :class="['badge', entryBadgeClass(entry.status)]">
                  {{ entry.status }}
                </span>
              </td>
              <td>{{ entry.message || '—' }}</td>
              <td>{{ formatTime(entry.lastSyncTime) }}</td>
            </tr>
          </tbody>
        </table>
      </Tab>
    </ResourceTabs>
  </div>
</template>

<script>
import ResourceTabs from '@shell/components/form/ResourceTabs';
import Tab          from '@shell/components/Tabbed/Tab';

export default {
  name: 'ManagedSecretDetail',

  components: { ResourceTabs, Tab },

  props: {
    value: { type: Object, required: true },
  },

  data() {
    return { activeTab: 'overview' };
  },

  computed: {
    syncEntries() {
      return this.value.status?.syncStatus || [];
    },

    syncBadgeClass() {
      const map = {
        Synced:  'bg-success',
        Failed:  'bg-error',
        Partial: 'bg-warning',
        Pending: 'bg-info',
      };

      return map[this.value.syncState] || 'bg-info';
    },
  },

  methods: {
    selectorDisplay(sel) {
      if (!sel) return '';
      const labels = sel.matchLabels || {};

      return Object.entries(labels)
        .map(([k, v]) => `${ k }=${ v }`)
        .join(', ');
    },

    entryBadgeClass(status) {
      const map = {
        Synced:  'bg-success',
        Failed:  'bg-error',
        Pending: 'bg-info',
      };

      return map[status] || 'bg-info';
    },

    formatTime(t) {
      if (!t) return '—';

      return new Date(t).toLocaleString();
    },
  },
};
</script>

<style scoped>
.overview {
  padding: 20px 0;
}

.detail-section {
  margin-bottom: 24px;
}

.detail-section h3 {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 10px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  opacity: 0.7;
}

.detail-row {
  display: flex;
  gap: 16px;
  padding: 6px 0;
  border-bottom: 1px solid var(--border);
}

.detail-label {
  width: 160px;
  font-weight: 500;
  flex-shrink: 0;
}

.badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  color: white;
  font-size: 12px;
  font-weight: 600;
}

.targets-table,
.sync-table {
  width: 100%;
  border-collapse: collapse;
}

.targets-table th,
.targets-table td,
.sync-table th,
.sync-table td {
  padding: 8px 12px;
  text-align: left;
  border-bottom: 1px solid var(--border);
}

.targets-table th,
.sync-table th {
  font-weight: 600;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  opacity: 0.7;
}

.no-entries {
  padding: 20px;
  text-align: center;
  opacity: 0.6;
}
</style>
