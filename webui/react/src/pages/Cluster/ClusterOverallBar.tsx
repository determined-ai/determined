import React, { useMemo } from 'react';

import Section from 'components/Section';
import SlotAllocationBar from 'components/SlotAllocationBar';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { useClusterStore } from 'stores/cluster';
import { ShirtSize } from 'themes';
import { ResourceType } from 'types';
import { getSlotContainerStates } from 'utils/cluster';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

export const ClusterOverallBar: React.FC = () => {
  const overview = Loadable.match(useObservable(useClusterStore().clusterOverview), {
    Loaded: (co) => co,
    NotLoaded: () => undefined,
  });
  const agents = Loadable.match(useObservable(useClusterStore().agents), {
    Loaded: (ag) => ag,
    NotLoaded: () => undefined,
  });

  const cudaSlotStates = useMemo(() => {
    return getSlotContainerStates(agents || [], ResourceType.CUDA);
  }, [agents]);

  const rocmSlotStates = useMemo(() => {
    return getSlotContainerStates(agents || [], ResourceType.ROCM);
  }, [agents]);

  const cpuSlotStates = useMemo(() => {
    return getSlotContainerStates(agents || [], ResourceType.CPU);
  }, [agents]);

  return (
    <Section hideTitle title="Overall Allocation">
      <Spinner conditionalRender spinning={agents === undefined || overview === undefined}>
        {agents !== undefined && overview !== undefined ? ( // This is ok as the Spinner has conditionalRender active
          <>
            {overview.CUDA.total + overview.ROCM.total + overview.CPU.total === 0 ? (
              <Message title="No connected agents." type={MessageType.Empty} />
            ) : null}
            {overview.CUDA.total > 0 && (
              <SlotAllocationBar
                resourceStates={cudaSlotStates}
                showLegends
                size={ShirtSize.Large}
                title={`Compute (${ResourceType.CUDA})`}
                totalSlots={overview.CUDA.total}
              />
            )}
            {overview.ROCM.total > 0 && (
              <SlotAllocationBar
                resourceStates={rocmSlotStates}
                showLegends
                size={ShirtSize.Large}
                title={`Compute (${ResourceType.ROCM})`}
                totalSlots={overview.ROCM.total}
              />
            )}
            {overview.CPU.total > 0 && (
              <SlotAllocationBar
                resourceStates={cpuSlotStates}
                showLegends
                size={ShirtSize.Large}
                title={`Compute (${ResourceType.CPU})`}
                totalSlots={overview.CPU.total}
              />
            )}
          </>
        ) : undefined}
      </Spinner>
    </Section>
  );
};
