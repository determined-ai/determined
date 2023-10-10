import React, { PropsWithChildren } from 'react';

import css from './ClusterTopology.module.scss';

interface NodeElementProps {
  name: string;
  slots: number;
  filledSlots: number;
}

const NodeElement: React.FC<PropsWithChildren<NodeElementProps>> = ({
  name,
  slots,
  filledSlots,
}) => {
  return (
    <div className={css.node}>
      <span className={css.nodeName}>{name}</span>
      <span className={css.nodeCluster}>
        {Array.from(Array(slots)).map((_, idx) => (
          <span
            className={`${css.nodeSlot} ${idx + 1 <= filledSlots ? css.filled : ''}`}
            key={`slot${idx}`}
          />
        ))}
      </span>
    </div>
  );
};

const ClusterTopology: React.FC<PropsWithChildren> = () => {
  return (
    <div className={css.container}>
      <NodeElement filledSlots={3} name="test123" slots={8} />
    </div>
  );
};

export default ClusterTopology;
