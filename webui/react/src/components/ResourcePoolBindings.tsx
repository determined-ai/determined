import { useObservable } from 'micro-observables';
import { useCallback, useEffect, useMemo } from 'react';

import Avatar, { Size } from 'components/kit/Avatar';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import { Loadable } from 'components/kit/utils/loadable';
import Link from 'components/Link';
import ResourcePoolBindingModalComponent from 'components/ResourcePoolBindingModal';
import { ColumnDef } from 'components/Table/InteractiveTable';
import ResponsiveTable from 'components/Table/ResponsiveTable';
import { paths } from 'routes/utils';
import clusterStore from 'stores/cluster';
import workspaceStore from 'stores/workspaces';
import { ResourcePool, Workspace } from 'types';
import { alphaNumericSorter } from 'utils/sort';

import css from './ResourcePoolBindings.module.scss';

interface Props {
  pool: ResourcePool;
}

const ResourcePoolBindings = ({ pool }: Props): JSX.Element => {
  const ResourcePoolBindingModal = useModal(ResourcePoolBindingModalComponent);
  const resourcePoolBindingMap = useObservable(clusterStore.resourcePoolBindings);
  const resourcePoolBindings: number[] = resourcePoolBindingMap.get(pool.name, []);
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));

  useEffect(() => {
    return clusterStore.fetchResourcePoolBindings(pool.name);
  }, [pool.name]);

  const tableColumns: ColumnDef<Workspace>[] = useMemo(() => {
    return [
      {
        dataIndex: 'workspace',
        defaultWidth: 100,
        key: 'workspace',
        render: (_, record) => (
          <div className={css.tableRow}>
            <Avatar size={Size.Medium} square text={record.name} textColor="black" />
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
  }, []);

  const tableRows = useMemo(() => {
    return workspaces.filter((w) => resourcePoolBindings.includes(w.id));
  }, [resourcePoolBindings, workspaces]);

  const onSaveBindings = useCallback(
    (bindings: string[]) => {
      const workspaceIds = workspaces.filter((w) => bindings.includes(w.name)).map((w) => w.id);
      clusterStore.overwriteResourcePoolBindings(pool.name, workspaceIds);
    },
    [workspaces, pool.name],
  );

  return (
    <>
      <div className={css.header}>
        <h5 style={{ margin: 'unset' }}>Bindings</h5>
        <Button
          disabled={pool.defaultAuxPool || pool.defaultComputePool}
          tooltip={
            pool.defaultAuxPool || pool.defaultComputePool
              ? 'Cannot bind default compute or aux pool'
              : ''
          }
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
        pool={pool.name}
        workspaces={workspaces.map((w) => w.name)}
        onSave={onSaveBindings}
      />
    </>
  );
};

export default ResourcePoolBindings;
