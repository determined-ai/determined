import Tooltip from 'hew/Tooltip';
import React, { PropsWithChildren, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { Resource, SlotsRecord } from 'types';

import css from './NodeElement.module.scss';

interface NodeElementProps {
  name: string;
  resources: Resource[];
  slots?: SlotsRecord;
  isRunning?: boolean;
}

const NodeElement: React.FC<PropsWithChildren<NodeElementProps>> = ({
  name,
  slots,
  resources,
  isRunning = true,
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
    (isActive: boolean) => {
      if (singleSlot) slotStyles.push(css.singleSlot);
      if (coupleSlot) slotStyles.push(css.coupleSlot);
      if (isActive) slotStyles.push(css.active);
      if (!isRunning) slotStyles.push(css.notRunning);

      return slotStyles.join(' ');
    },

    // eslint-disable-next-line react-hooks/exhaustive-deps
    [isRunning, singleSlot, coupleSlot],
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
          <span className={getSlotStyles(container !== undefined)} key={`slot${idx}`} />
        ))}
      </span>
    </div>
  );
};

export default NodeElement;
