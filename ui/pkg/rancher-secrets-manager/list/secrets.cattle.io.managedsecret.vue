<template>
  <ResourceTable
    v-bind="$attrs"
    :schema="schema"
    :rows="rows"
    :headers="headers"
    key-field="_key"
  >
    <template #cell:syncState="{ row }">
      <span :class="['badge', syncBadgeClass(row)]">
        {{ row.syncState }}
      </span>
    </template>
  </ResourceTable>
</template>

<script>
import ResourceTable from '@shell/components/ResourceTable';

export default {
  name: 'ManagedSecretList',

  components: { ResourceTable },

  props: {
    schema: { type: Object, required: true },
    rows:   { type: Array,  required: true },
  },

  computed: {
    headers() {
      return [
        {
          name:  'name',
          label: 'Name',
          value: 'nameDisplay',
          sort:  ['nameSort'],
          width: 220,
        },
        {
          name:  'sourceSecret',
          label: 'Source Secret',
          value: 'sourceSecretDisplay',
          sort:  ['sourceSecretDisplay'],
        },
        {
          name:  'targetCount',
          label: 'Targets',
          value: 'targetCount',
          sort:  ['targetCount'],
          width: 80,
          align: 'center',
        },
        {
          name:  'syncedCount',
          label: 'Synced',
          value: 'syncedCount',
          sort:  ['syncedCount'],
          width: 80,
          align: 'center',
        },
        {
          name:  'syncState',
          label: 'Status',
          value: 'syncState',
          sort:  ['syncState'],
          width: 110,
        },
        {
          name:  'age',
          label: 'Age',
          value: 'creationTimestamp',
          sort:  ['creationTimestamp'],
          width: 80,
        },
      ];
    },
  },

  methods: {
    syncBadgeClass(row) {
      const map = {
        Synced:  'bg-success',
        Failed:  'bg-error',
        Partial: 'bg-warning',
        Pending: 'bg-info',
        Paused:  'bg-muted',
      };

      return map[row.syncState] || 'bg-info';
    },
  },
};
</script>

<style scoped>
.badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  color: white;
  font-size: 12px;
  font-weight: 600;
}
</style>
