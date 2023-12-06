import Tooltip from 'hew/Tooltip';
import React, { PropsWithChildren, useEffect, useMemo, useRef, useState } from 'react';

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
  const [containerWidth, setContainerWidth] = useState(0);
  const shouldTruncate = useMemo(() => name.length > 5, [name]);
  const slotsContainer = useRef<HTMLSpanElement>(null);
  const singleSlot = slots.length === 1;
  const fewSlot = slots.length === 2;
  const styles = [css.nodeSlot];

  if (singleSlot) styles.push(css.singleSlot);
  if (fewSlot) styles.push(css.fewSlot);

  useEffect(() => {
    setContainerWidth(slotsContainer.current?.getBoundingClientRect().width || 0);
  }, []);

  return (
    <div className={css.node}>
      {shouldTruncate ? (
        <Tooltip content={name}>
          <span className={css.nodeName} style={{ maxWidth: containerWidth }}>
            {name}
          </span>
        </Tooltip>
      ) : (
        <span className={css.nodeName}>{name}</span>
      )}
      <span className={css.nodeCluster} ref={slotsContainer}>
        {slots.map(({ enabled }, idx) => (
          <span className={`${styles.join(' ')} ${enabled ? css.active : ''}`} key={`slot${idx}`} />
        ))}
      </span>
    </div>
  );
};

const Topology: React.FC<PropsWithChildren<Props>> = ({ nodes }) => {
  return (
    <Section title="Topology">
      <div className={`${css.mainContainer} ${css.nodesContainer}`}>
        {nodes.map(({ id, resources }) => {
          return <NodeElement key={id} name={id} slots={resources} />;
        })}
      </div>
    </Section>
  );
};

export default Topology;
