import SteveModel from '@shell/plugins/steve/steve-class';

export default class ManagedSecret extends SteveModel {
  get sourceSecretDisplay() {
    const ref = this.spec?.secretRef;

    return ref ? `${ ref.namespace }/${ ref.name }` : '';
  }

  get targetCount() {
    return this.status?.targetCount || 0;
  }

  get syncedCount() {
    return this.status?.syncedCount || 0;
  }

  get syncState() {
    const total  = this.targetCount;
    const synced = this.syncedCount;

    if (!total)           return 'Pending';
    if (synced === total) return 'Synced';
    if (synced === 0)     return 'Failed';

    return 'Partial';
  }

  get stateLabel() {
    return this.syncState;
  }

  get stateColor() {
    const map = {
      Synced:  'text-success',
      Failed:  'text-error',
      Partial: 'text-warning',
      Pending: 'text-info',
    };

    return map[this.syncState] || 'text-info';
  }
}
