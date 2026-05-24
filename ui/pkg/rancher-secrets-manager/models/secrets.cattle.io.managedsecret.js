export default {
  computed: {
    sourceSecretDisplay() {
      const ref = this.spec?.secretRef;

      return ref ? `${ ref.namespace }/${ ref.name }` : '';
    },

    targetCount() {
      return this.status?.targetCount || 0;
    },

    syncedCount() {
      return this.status?.syncedCount || 0;
    },

    syncState() {
      const total  = this.targetCount;
      const synced = this.syncedCount;

      if (!total)          return 'Pending';
      if (synced === total) return 'Synced';
      if (synced === 0)     return 'Failed';

      return 'Partial';
    },

    stateLabel() {
      return this.syncState;
    },

    stateColor() {
      const map = {
        Synced:  'text-success',
        Failed:  'text-error',
        Partial: 'text-warning',
        Pending: 'text-info',
      };

      return map[this.syncState] || 'text-info';
    },
  },
};
