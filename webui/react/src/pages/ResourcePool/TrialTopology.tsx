import Tooltip from 'hew/Tooltip';
import { range } from 'lodash';
import React, { PropsWithChildren, useEffect, useMemo, useRef, useState } from 'react';

import Section from 'components/Section';
import { Agent } from 'types';

import css from './TrialTopology.module.scss';

interface NodeElementProps {
  name: string;
  numOfSlots: number;
}

interface Props {
  nodes: Agent[];
}

export const NodeElement: React.FC<PropsWithChildren<NodeElementProps>> = ({
  name,
  numOfSlots,
}) => {
  const [containerWidth, setContainerWidth] = useState(0);
  const shouldTruncate = useMemo(() => name.length > 5, [name]);
  const slotsContainer = useRef<HTMLSpanElement>(null);
  const singleSlot = numOfSlots === 1;
  const coupleSlot = numOfSlots === 2;
  const styles = [css.nodeSlot];

  if (singleSlot) styles.push(css.singleSlot);
  if (coupleSlot) styles.push(css.coupleSlot);

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
        {range(numOfSlots).map((idx) => (
          <span className={`${styles.join(' ')} ${css.active}`} key={`slot${idx}`} />
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
          return (
            <NodeElement
              key={id}
              name={id}
              numOfSlots={(slots !== undefined ? Object.values(slots) : resources).length}
            />
          );
        })}
      </div>
    </Section>
  );
};

export default Topology;
