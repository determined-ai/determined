import Tooltip from 'hew/Tooltip';
import React, { PropsWithChildren, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { Resource, SlotsRecord } from 'types';

import css from './NodeElement.module.scss';

interface NodeElementProps {
  name: string;
  resources: Resource[];
  slots?: SlotsRecord;
  isRunning?: boolean;
  limitSlots?: number;
}

const NodeElement: React.FC<PropsWithChildren<NodeElementProps>> = ({
  name,
  slots,
  resources,
  isRunning = true,
  limitSlots = 0,
}) => {
  const [containerWidth, setContainerWidth] = useState(0);
  const shouldTruncate = useMemo(() => name.length > 5, [name]);
  const slotsContainer = useRef<HTMLSpanElement>(null);
  const slotsData = useMemo(
    () => (slots !== undefined ? Object.values(slots) : resources),
    [slots, resources],
  );
  const singleSlot = slotsData.length === 1;
  const coupleSlot = slotsData.length === 2;
  const slotStyles = [css.nodeSlot];
  const nodeStyles = [css.node];
  const nodeClusterStyles = [css.nodeCluster];

  if (!isRunning) {
    nodeStyles.push(css.notRunning);
    nodeClusterStyles.push(css.notRunning);
  }

  const getSlotStyles = useCallback(
    (isActive: boolean, index: number) => {
      if (singleSlot) slotStyles.push(css.singleSlot);
      if (coupleSlot) slotStyles.push(css.coupleSlot);
      if (isActive) slotStyles.push(css.active);
      if (!isRunning) slotStyles.push(css.notRunning);
      if (limitSlots !== 0) {
        if (index + 1 > limitSlots && isActive) slotStyles.push(css.limitedActive); // it means that there we're visualizing the node where only part of the active slots are relevant to the UI context
      }

      return slotStyles.join(' ');
    },

    // eslint-disable-next-line react-hooks/exhaustive-deps
    [isRunning, singleSlot, coupleSlot, limitSlots],
  );

  useEffect(() => {
    setContainerWidth(slotsContainer.current?.getBoundingClientRect().width || 0);
  }, []);

  return (
    <div className={nodeStyles.join(' ')}>
      {shouldTruncate ? (
        <Tooltip content={name}>
          <span className={css.nodeName} style={{ maxWidth: containerWidth }}>
            {name}
          </span>
        </Tooltip>
      ) : (
        <span className={css.nodeName}>{name}</span>
      )}
      <span className={nodeClusterStyles.join(' ')} ref={slotsContainer}>
        {slotsData.map(({ container }, idx) => (
          <span className={getSlotStyles(container !== undefined, idx)} key={`slot${idx}`} />
        ))}
      </span>
    </div>
  );
};

export default NodeElement;
