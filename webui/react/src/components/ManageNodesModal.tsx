import React, { useCallback, useState } from 'react';

import Icon from 'hew/Icon';
import Input from 'hew/Input';
import Row from 'hew/Row';
import { Modal } from 'hew/Modal';
import Toggle from 'hew/Toggle';
import { Body, Label } from 'hew/Typography';
import { Agent, Resource, ResourceType } from 'types';
import handleError from 'utils/error';

import css from './ManageNodesModal.module.scss';

interface Props {
  nodes: Agent[];
}

const defaultNodes: Agent[] = [
  { id: 'ABC', registeredTime: 1000, resourcePools: [], resources: [
    { id: 'A', type: ResourceType.UNSPECIFIED, name: 'A', enabled: true },
  ]},
  { id: 'DEF', registeredTime: 2000, resourcePools: [], resources: [] },
];

const ManageNodesModalComponent = ({ nodes = defaultNodes }: Props): JSX.Element => {
  const [searchText, setSearchText] = useState<string>('');
  const [toggleStates, setToggleStates] = useState<Record<string, boolean>>(
    nodes.reduce((obj: Record<string, boolean>, node: Agent) => {
      node.resources.forEach((r: Resource) => {
        obj[r.id] = r.enabled;
      });
      return obj;
    }, {})
  );

  const onFilterChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchText(e.target.value);
  }, []);

  const onToggleNode = useCallback((nodeId: string) => {
    setToggleStates((prev) => ({ ...prev, [nodeId]: !prev[nodeId] }));
  }, []);

  const onFormSubmit = useCallback(() => {

  }, []);

  return (
    <Modal
      cancel
      submit={{
        disabled: false,
        text: "Apply Changes",
        handler: onFormSubmit,
        handleError,
      }}
      size="small"
      title="Manage Nodes">
      <div className={css.content}>
        <Body>Disable nodes to make them temporarily unavailable for job assignment.</Body>
        <Label>Node Availability</Label>
        <Input
          autoFocus
          placeholder="Filter nodes"
          prefix={<Icon name="search" title="Search" />}
          value={searchText}
          onChange={onFilterChange}
        />
        {nodes.map((node) => (
          <Row key={node.id}>
            <Toggle checked={toggleStates[node.id]} onChange={() => onToggleNode(node.id)} />
            {node.id}
          </Row>
        ))}
      </div>
    </Modal>
  );
};

export default ManageNodesModalComponent;
