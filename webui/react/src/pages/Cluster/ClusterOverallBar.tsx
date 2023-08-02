import React, { useMemo } from 'react';

import Message, { MessageType } from 'components/Message';
import Section from 'components/Section';
import SlotAllocationBar from 'components/SlotAllocationBar';
import Spinner from 'components/Spinner';
import clusterStore from 'stores/cluster';
import { ShirtSize } from 'themes';
import { ResourceType } from 'types';
import { getSlotContainerStates } from 'utils/cluster';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

export const ClusterOverallBar: React.FC = () => {
  const overview = useObservable(clusterStore.clusterOverview);
  const agents = useObservable(clusterStore.agents);

  const cudaSlotStates = useMemo(() => {
    return getSlotContainerStates(
      (Loadable.isLoaded(agents) && agents.data) || [],
      ResourceType.CUDA,
    ); // can't use a const here due to types
  }, [agents]);

  const rocmSlotStates = useMemo(() => {
    return getSlotContainerStates(
      (Loadable.isLoaded(agents) && agents.data) || [],
      ResourceType.ROCM,
    );
  }, [agents]);

  const cpuSlotStates = useMemo(() => {
    return getSlotContainerStates(
      (Loadable.isLoaded(agents) && agents.data) || [],
      ResourceType.CPU,
    );
  }, [agents]);

  return (
    <Section hideTitle title="Overall Allocation">
      <Spinner
        conditionalRender
        spinning={Loadable.isLoading(agents) || Loadable.isLoading(overview)}>
        {Loadable.isLoaded(agents) && Loadable.isLoaded(overview) ? ( // This is ok as the Spinner has conditionalRender active
          <>
            {overview.data.CUDA.total + overview.data.ROCM.total + overview.data.CPU.total === 0 ? (
              <Message title="No connected agents." type={MessageType.Empty} />
            ) : null}
            {overview.data.CUDA.total > 0 && (
              <SlotAllocationBar
                resourceStates={cudaSlotStates}
                showLegends
                size={ShirtSize.Large}
                title={`Compute (${ResourceType.CUDA})`}
                totalSlots={overview.data.CUDA.total}
              />
            )}
            {overview.data.ROCM.total > 0 && (
              <SlotAllocationBar
                resourceStates={rocmSlotStates}
                showLegends
                size={ShirtSize.Large}
                title={`Compute (${ResourceType.ROCM})`}
                totalSlots={overview.data.ROCM.total}
              />
            )}
            {overview.data.CPU.total > 0 && (
              <SlotAllocationBar
                resourceStates={cpuSlotStates}
                showLegends
                size={ShirtSize.Large}
                title={`Compute (${ResourceType.CPU})`}
                totalSlots={overview.data.CPU.total}
              />
            )}
          </>
        ) : undefined}
      </Spinner>
    </Section>
  );
};
