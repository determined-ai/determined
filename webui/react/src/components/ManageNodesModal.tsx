import Icon from 'hew/Icon';
import Input from 'hew/Input';
import Message from 'hew/Message';
import { Modal } from 'hew/Modal';
import Row from 'hew/Row';
import Toggle from 'hew/Toggle';
import { Body, Label } from 'hew/Typography';
import { isEqual } from 'lodash';
import React, { useCallback, useMemo, useState } from 'react';

import { disableAgent, enableAgent } from 'services/api';
import { Agent } from 'types';
import handleError from 'utils/error';

import css from './ManageNodesModal.module.scss';

interface Props {
  nodes: Agent[];
}

// const testNodes: Agent[] = [
//   { id: 'ABC', enabled: true, registeredTime: 1000, resourcePools: [], resources: []},
//   { id: 'DEF', enabled: false, registeredTime: 2000, resourcePools: [], resources: [] },
// ];

const ManageNodesModalComponent = ({ nodes }: Props): JSX.Element => {
  // nodes = testNodes;
  const originalNodes = useMemo(
    () =>
      nodes.reduce((obj: Record<string, boolean>, node: Agent) => {
        obj[node.id] = !!node.enabled;
        return obj;
      }, {}),
    [nodes],
  );

  const [searchText, setSearchText] = useState<string>('');
  const [toggleStates, setToggleStates] = useState<Record<string, boolean>>(originalNodes);

  const onFilterChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchText(e.target.value);
  }, []);

  const onToggleNode = useCallback((nodeId: string) => {
    setToggleStates((prev) => ({ ...prev, [nodeId]: !prev[nodeId] }));
  }, []);

  const onFormSubmit = useCallback(async () => {
    for (const nodeId in toggleStates) {
      if (toggleStates[nodeId] !== originalNodes[nodeId]) {
        if (toggleStates[nodeId]) {
          await enableAgent(nodeId);
        } else {
          await disableAgent(nodeId);
        }
      }
    }
  }, [originalNodes, toggleStates]);

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled: isEqual(originalNodes, toggleStates),
        handleError,
        handler: onFormSubmit,
        text: 'Apply Changes',
      }}
      title="Manage Nodes">
      <div className={css.content}>
        <Body>Disable nodes to make them temporarily unavailable for job assignment.</Body>
        <Label>Node Availability</Label>
        {nodes.length >= 10 && (
          <Input
            autoFocus
            placeholder="Filter nodes"
            prefix={<Icon name="search" title="Search" />}
            value={searchText}
            onChange={onFilterChange}
          />
        )}
        {nodes.length === 0 && <Message title="No active agents." />}
        {nodes
          .filter((node) => !searchText.trim() || node.id.includes(searchText))
          .map((node) => (
            <Row height={35} key={node.id}>
              <Toggle checked={toggleStates[node.id]} onChange={() => onToggleNode(node.id)} />
              <Label>{node.id}</Label>
            </Row>
          ))}
      </div>
    </Modal>
  );
};

export default ManageNodesModalComponent;
