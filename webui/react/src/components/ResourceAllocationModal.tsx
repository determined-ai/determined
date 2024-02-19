import Button from 'hew/Button';
import Link from 'hew/Link';
import { Modal } from 'hew/Modal';
import { Loadable } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useState } from 'react';

import NodeElement from 'pages/ResourcePool/NodeElement';
import { handlePath, paths } from 'routes/utils';
import clusterStore from 'stores/cluster';
import { Agent, Resource } from 'types';
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
  const [limitSlots, setLimitSlots] = useState(0);
  const [activeNodes, setActiveNodes] = useState<Agent[]>([]);
  const agents = Loadable.waitFor(useObservable(clusterStore.agents));
  const footer = (
    <div className={css.footer}>
      <Button onClick={onClose}>Done</Button>
    </div>
  );
  const renderNodes = useCallback(() => {
    if (activeNodes.length > 0) {
      return activeNodes.map(({ id, resources, slots }, idx) => (
        <NodeElement
          key={`${idx}${id}`}
          limitSlots={limitSlots}
          name={id}
          resources={resources}
          slots={slots}
        />
      ));
    }

    return nodes.map(({ nodeName, slotsIds }, idx) => (
      <NodeElement
        isRunning={false}
        key={`${idx}${nodeName}`}
        name={nodeName}
        resources={slotsIds}
      />
    ));
  }, [activeNodes, limitSlots, nodes]);

  useEffect(() => {
    if (isRunning && agents.length !== 0) {
      const experimentNodes: Agent[] = [];

      for (const node of nodes) {
        const agent = agents.find((agent) => agent.id === node.nodeName);
        if (agent !== undefined) {
          experimentNodes.push(agent);

          const usedSlots = node.slotsIds.filter((slot) => slot.container !== undefined).length;
          const activeAgentSlots = agent.resources.filter(
            (slot) => slot.container !== undefined,
          ).length;

          if (usedSlots !== activeAgentSlots) setLimitSlots(usedSlots);
        }
      }

      setActiveNodes(experimentNodes);
    } else {
      setActiveNodes([]);
      setLimitSlots(0);
    }
  }, [agents, nodes, isRunning]);

  return (
    <Modal cancel footer={footer} size="medium" title="Resource Allocation">
      <div className={css.base}>
        <div className={css.nodesContainer}>{renderNodes()}</div>
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
