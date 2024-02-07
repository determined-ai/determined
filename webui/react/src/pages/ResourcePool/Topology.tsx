import React, { PropsWithChildren } from 'react';

import Section from 'components/Section';
import NodeElement from 'pages/ResourcePool/NodeElement';
import { Agent } from 'types';

import css from './Topology.module.scss';

interface Props {
  nodes: Agent[];
}

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
