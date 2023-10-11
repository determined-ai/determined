import React, { PropsWithChildren } from 'react';

import Section from 'components/Section';
import { Agent, Resource } from 'types';

import css from './Topology.module.scss';

interface NodeElementProps {
  name: string;
  slots: Resource[];
}

interface Props {
  nodes: Agent[];
}

const NodeElement: React.FC<PropsWithChildren<NodeElementProps>> = ({ name, slots }) => {
  const singleSlot = slots.length === 1;
  const fewSlot = slots.length === 2;
  const styles = [css.nodeSlot];

  if (singleSlot) styles.push(css.singleSlot);
  if (fewSlot) styles.push(css.fewSlot);

  return (
    <div className={css.node}>
      <span className={css.nodeName}>{name}</span>
      <span className={css.nodeCluster}>
        {slots.map(({ enabled }, idx) => (
          <span className={`${styles.join(' ')} ${enabled ? css.active : ''}`} key={`slot${idx}`} />
        ))}
      </span>
    </div>
  );
};

const Topology: React.FC<PropsWithChildren<Props>> = ({ nodes }) => {
  return (
    <>
      {nodes.length ? (
        <Section className={css.mainContainer} title="Topology">
          <div className={css.nodesContainer}>
            {nodes.map(({ id, resources }) => {
              const slots = resources.length;

              return slots ? <NodeElement key={id} name={id} slots={resources} /> : null;
            })}
          </div>
        </Section>
      ) : null}
    </>
  );
};

export default Topology;
