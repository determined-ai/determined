import Tooltip from 'hew/Tooltip';
import React, { PropsWithChildren, useEffect, useMemo, useRef, useState } from 'react';

import Section from 'components/Section';
import { Agent, Resource, Slot } from 'types';

import css from './Topology.module.scss';

interface NodeElementProps {
  name: string;
  resources: Resource[];
  slots?: Slot;
}

interface Props {
  nodes: Agent[];
}

const NodeElement: React.FC<PropsWithChildren<NodeElementProps>> = ({ name, slots, resources }) => {
  const [containerWidth, setContainerWidth] = useState(0);
  const shouldTruncate = useMemo(() => name.length > 5, [name]);
  const slotsContainer = useRef<HTMLSpanElement>(null);
  const slotsData = useMemo(
    () => (slots !== undefined ? Object.values(slots) : resources),
    [slots, resources],
  );
  const singleSlot = slotsData.length === 1;
  const fewSlot = slotsData.length === 2;
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
        {slotsData.map(({ enabled }, idx) => (
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
        {nodes.map(({ id, resources, slots }) => {
          return <NodeElement key={id} name={id} resources={resources} slots={slots} />;
        })}
      </div>
    </Section>
  );
};

export default Topology;
