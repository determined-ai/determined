import React, { useMemo } from 'react';

import Grid from 'components/Grid';
import Message from 'components/Message';
import Page from 'components/Page';
import ResourceChart from 'components/ResourceChart';
import Spinner from 'components/Spinner';
import AgentsCtx from 'contexts/Agents';
import { Resource, ResourceType } from 'types';
import { categorize } from 'utils/data';

const Cluster: React.FC = () => {
  const agents = AgentsCtx.useStateContext();

  const availableResources = useMemo(() => {
    if (!agents.data) return {};
    const resourceList = agents.data
      .map(agent => agent.resources)
      .reduce((acc: Resource[], resources: Resource[]) => {
        acc.push(...resources);
        return resources;
      }, []);
    return categorize(resourceList, (res: Resource) => res.type);

  }, [ agents ]);

  const availableResourceTypes = Object.keys(availableResources);

  let unhappyView: React.ReactNode = null;

  if (!agents.data) {
    unhappyView = <Spinner />;
  } else if (agents.data.length === 0) {
    unhappyView = (<Message>
      No agents connected.
    </Message>);
  } else if (availableResourceTypes.length === 0) {
    unhappyView = (<Message>
      No slots available.
    </Message>);
  }

  return (
    <Page title="Cluster">
      {unhappyView ? unhappyView :
        <Grid minItemWidth={50}>
          {Object.values(ResourceType).map((resourceType, idx) => (
            <ResourceChart key={idx}
              resources={availableResources[resourceType]}
              title={resourceType + 's'} />
          ))}
        </Grid>
      }
    </Page>
  );
};

export default Cluster;
