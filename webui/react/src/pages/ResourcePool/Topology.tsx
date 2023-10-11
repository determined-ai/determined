import { useObservable } from 'micro-observables';
import React, { PropsWithChildren } from 'react';

import { Loadable } from 'components/kit/utils/loadable';
import Section from 'components/Section';
import clusterStore from 'stores/cluster';

import css from './Topology.module.scss';

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
  const singleSlot = slots === 1;
  const fewSlot = slots === 2;

  return (
    <div className={css.node}>
      <span className={css.nodeName}>{name}</span>
      <span className={css.nodeCluster}>
        {Array.from(Array(slots)).map((_, idx) => (
          <span
            className={`${css.nodeSlot} ${idx + 1 <= enabledSlots ? css.active : ''} ${
              singleSlot ? css.singleSlot : ''
            } ${fewSlot ? css.fewSlot : ''}`}
            key={`slot${idx}`}
          />
        ))}
      </span>
    </div>
  );
};

const Topology: React.FC<PropsWithChildren> = () => {
  const nodes = Loadable.waitFor(useObservable(clusterStore.agents));

  return (
    <>
      {nodes.length ? (
        <Section className={css.mainContainer} title="Topology">
          <div className={css.nodesContainer}>
            {nodes.map(({ id, resources }) => {
              const enabledSlots = resources.reduce(
                (acc, { enabled }) => (enabled ? acc++ : acc),
                0,
              );
              const slots = resources.length;

              return <NodeElement enabledSlots={enabledSlots} key={id} name={id} slots={slots} />;
            })}
          </div>
        </Section>
      ) : null}
    </>
  );
};

export default Topology;
