import Button from 'hew/Button';
import { Modal } from 'hew/Modal';
import React from 'react';

import NodeElement from 'pages/ResourcePool/NodeElement';
import { Resource } from 'types';

import css from './ResourceAllocationModal.module.scss';

interface Node {
  nodeName: string;
  slotsIds: Resource[];
}

export interface Props {
  rpName: string;
  accelerator: string;
  nodes: Node[];
  onClose: () => void;
  isRunning: boolean;
}

const ResourceAllocationModalComponent: React.FC<Props> = ({
  rpName,
  nodes,
  accelerator,
  isRunning,
  onClose,
}: Props) => {
  const footer = (
    <div className={css.footer}>
      <Button onClick={onClose}>Done</Button>
    </div>
  );

  return (
    <Modal cancel footer={footer} size="medium" title="Resource Allocation">
      <div className={css.base}>
        <div className={css.nodesContainer}>
          {nodes.map(({ nodeName, slotsIds }, idx) => (
            <NodeElement
              isRunning={isRunning}
              key={`${idx}${nodeName}`}
              name={nodeName}
              resources={slotsIds}
            />
          ))}
        </div>
        <div className={css.infoContainer}>
          <span>Resource Pool</span>
          <span className={css.dots} />
          <span>{rpName}</span>
        </div>
        <div className={css.infoContainer}>
          <span>Accelerator</span>
          <span className={css.dots} />
          <span>{accelerator}</span>
        </div>
        {nodes.map(({ nodeName, slotsIds }) => (
          <div className={css.slotsContainer} key={`info_${nodeName}`}>
            <div className={css.infoContainer}>
              <span>Node ID</span>
              <span className={css.dots} />
              <span>{nodeName}</span>
            </div>
            {slotsIds.map((id, idx) => (
              <div className={css.infoContainer} key={`slot_${id}`}>
                <span>Slot {idx + 1} ID</span>
                <span className={css.dots} />
                <span>{id.container?.id}</span>
              </div>
            ))}
          </div>
        ))}
      </div>
    </Modal>
  );
};

export default ResourceAllocationModalComponent;
