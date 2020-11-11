import React, { useMemo } from 'react';

import Grid from 'components/Grid';
import Message from 'components/Message';
import Page from 'components/Page';
import ResourceChart from 'components/ResourceChart';
import Spinner from 'components/Spinner';
import Agents from 'contexts/Agents';
import { Resource, ResourceType } from 'types';
import { categorize } from 'utils/data';

const Cluster: React.FC = () => {
  const agents = Agents.useStateContext();

  const availableResources = useMemo(() => {
    if (!agents.data) return {};
    const resourceList = agents.data
      .map(agent => agent.resources)
      .flat()
      .filter(resource => resource.enabled);
    return categorize(resourceList, (res: Resource) => res.type);
  }, [ agents ]);

  const availableResourceTypes = Object.keys(availableResources);

  if (!agents.data) {
    return <Spinner />;
  } else if (agents.data.length === 0) {
    return <Message title="No Agents connected" />;
  } else if (availableResourceTypes.length === 0) {
    return <Message title="No Slots available" />;
  }

  return (
    <Page id="cluster" title="Cluster">
      <Grid minItemWidth={50}>
        {Object.values(ResourceType)
          .filter((resourceType) => (
            resourceType !== ResourceType.UNSPECIFIED
          ))
          .map((resourceType, idx) => (
            <ResourceChart
              key={idx}
              resources={availableResources[resourceType]}
              title={resourceType + 's'} />
          ))
        }
      </Grid>
    </Page>
  );
};

export default Cluster;
