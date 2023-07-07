import { useObservable } from 'micro-observables';
import { useCallback } from 'react';

import DynamicIcon from 'components/DynamicIcon';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import Link from 'components/Link';
import ResourcePoolBindingModalComponent from 'components/ResourcePoolBindingModal';
import { ColumnDef } from 'components/Table/InteractiveTable';
import ResponsiveTable from 'components/Table/ResponsiveTable';
import { paths } from 'routes/utils';
import clusterStore from 'stores/cluster';
import workspaceStore from 'stores/workspaces';
import { Workspace } from 'types';
import { Loadable } from 'utils/loadable';
import { alphaNumericSorter } from 'utils/sort';

import css from './ResourcePoolBindings.module.scss';

interface Props {
  poolName: string;
}

const ResourcePoolBindings = ({ poolName }: Props): JSX.Element => {
  const ResourcePoolBindingModal = useModal(ResourcePoolBindingModalComponent);
  const resourcePoolBindingMap = useObservable(clusterStore.resourcePoolBindings);
  const resourcePoolBindings: number[] = resourcePoolBindingMap.get(poolName, []);
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));

  const tableColumns: ColumnDef<Workspace>[] = [
    {
      dataIndex: 'workspace',
      defaultWidth: 100,
      key: 'workspace',
      render: (_, record) => (
        <div className={css.tableRow}>
          <DynamicIcon name={record.name} size={40} style={{ borderRadius: '100%' }} />
          <div>
            <div>
              <Link path={paths.workspaceDetails(record.id)}>{record.name}</Link>
            </div>
            <div className={css.numProjects}>
              {record.numProjects} {record.numProjects > 1 ? 'projects' : 'project'}
            </div>
          </div>
        </div>
      ),
      sorter: (a, b) => alphaNumericSorter(a.name, b.name),
      title: 'Workspace name',
    },
  ];

  const tableRows = workspaces.filter((w) => resourcePoolBindings.includes(w.id));

  const onSaveBindings = useCallback(
    (bindings: string[]) => {
      const workspaceIds = workspaces.filter((w) => bindings.includes(w.name)).map((w) => w.id);
      clusterStore.overwriteResourcePoolBindings(poolName, workspaceIds);
    },
    [workspaces, poolName],
  );

  return (
    <>
      <div className={css.header}>
        <h5 style={{ margin: 'unset' }}>Bindings</h5>
        <Button
          onClick={() => {
            ResourcePoolBindingModal.open();
          }}>
          Manage Bindings
        </Button>
      </div>
      <div>
        <ResponsiveTable columns={tableColumns} dataSource={tableRows} rowKey="id" size="small" />
      </div>
      <ResourcePoolBindingModal.Component
        bindings={workspaces.filter((w) => resourcePoolBindings.includes(w.id)).map((w) => w.name)}
        pool={poolName}
        workspaces={workspaces.map((w) => w.name)}
        onSave={onSaveBindings}
      />
    </>
  );
};

export default ResourcePoolBindings;
