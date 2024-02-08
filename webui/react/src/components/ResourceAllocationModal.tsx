import Button from 'hew/Button';
import Link from 'hew/Link';
import { Modal } from 'hew/Modal';
import React from 'react';

import NodeElement from 'pages/ResourcePool/NodeElement';
import { handlePath, paths } from 'routes/utils';
import { Resource } from 'types';
import { AnyMouseEvent } from 'utils/routes';

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

interface InfoContainerProps {
  label: string;
  info?: JSX.Element | string;
}

const InfoContainer: React.FC<InfoContainerProps> = ({ label, info }) => {
  return (
    <div className={css.infoContainer}>
      <span>{label}</span>
      <span className={css.dots} />
      <span>{info}</span>
    </div>
  );
};

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
        <InfoContainer
          info={
            <Link
              onClick={(e: AnyMouseEvent) => handlePath(e, { path: paths.resourcePool(rpName) })}>
              {rpName}
            </Link>
          }
          label="Resource Pool"
        />
        <InfoContainer info={accelerator} label="Accelerator" />
        {nodes.map(({ nodeName, slotsIds }) => (
          <div className={css.slotsContainer} key={`info_${nodeName}`}>
            <InfoContainer info={nodeName} label="Node ID" />
            {slotsIds.map((id, idx) => (
              <InfoContainer
                info={id.container?.id}
                key={`slot_${id}`}
                label={`Slot ${idx + 1} ID`}
              />
            ))}
          </div>
        ))}
      </div>
    </Modal>
  );
};

export default ResourceAllocationModalComponent;
