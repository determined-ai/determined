import { useObservable } from 'micro-observables';
import React, { PropsWithChildren } from 'react';

import { Loadable } from 'components/kit/utils/loadable';
import clusterStore from 'stores/cluster';

import css from './ClusterTopology.module.scss';

interface NodeElementProps {
  name: string;
  slots: number;
  enabledSlots: number;
}

const NodeElement: React.FC<PropsWithChildren<NodeElementProps>> = ({
  name,
  slots,
  enabledSlots,
}) => {
  return (
    <div className={css.node}>
      <span className={css.nodeName}>{name}</span>
      <span className={css.nodeCluster}>
        {Array.from(Array(slots)).map((_, idx) => (
          <span
            className={`${css.nodeSlot} ${idx + 1 <= enabledSlots ? css.filled : ''}`}
            key={`slot${idx}`}
          />
        ))}
      </span>
    </div>
  );
};

const ClusterTopology: React.FC<PropsWithChildren> = () => {
  const nodes = Loadable.waitFor(useObservable(clusterStore.agents));

  return (
    <div className={css.container}>
      {nodes.map(({ id, resources }) => {
        const enabledSlots = resources.reduce((acc, { enabled }) => (enabled ? acc++ : acc), 0);
        const slots = resources.length;

        return <NodeElement enabledSlots={enabledSlots} key={id} name={id} slots={slots} />;
      })}
    </div>
  );
};

export default ClusterTopology;
