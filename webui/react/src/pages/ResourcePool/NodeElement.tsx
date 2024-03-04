import Tooltip from 'hew/Tooltip';
import React, { PropsWithChildren, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { Resource, ResourceContainer, SlotsRecord } from 'types';

import css from './NodeElement.module.scss';

export type SimplifiedNode = { container?: Pick<ResourceContainer, 'id'> };
interface NodeElementProps {
  name: string;
  resources: Resource[] | SimplifiedNode[];
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
  const shouldTruncate = name.length > 5;
  const slotsContainer = useRef<HTMLSpanElement>(null);
  const slotsData = useMemo(
    () => (slots !== undefined ? Object.values(slots) : resources),
    [slots, resources],
  );
  const nodeStyles = [css.node];
  const nodeClusterStyles = [css.nodeCluster];

  if (!isRunning) {
    nodeStyles.push(css.notRunning);
    nodeClusterStyles.push(css.notRunning);
  }

  const getSlotStyles = useCallback(
    (isActive: boolean, index: number) => {
      const slotStyles = [css.nodeSlot];
      if (slotsData.length === 1) slotStyles.push(css.singleSlot);
      if (slotsData.length === 2) slotStyles.push(css.coupleSlot);
      if (!isRunning) slotStyles.push(css.notRunning);
      if (isActive) {
        slotStyles.push(css.active);
      } else {
        slotStyles.push(css.emptySlot);
      }
      if (limitSlots !== 0) {
        if (index + 1 > limitSlots && isActive) slotStyles.push(css.limitedActive); // it means that there we're visualizing the node where only part of the active slots are relevant to the UI context
      }

      return slotStyles.join(' ');
    },
    [isRunning, limitSlots, slotsData],
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
          <span className={getSlotStyles(container !== undefined, idx)} key={`${name}${idx}`} />
        ))}
      </span>
    </div>
  );
};

export default NodeElement;
