import React, { useMemo } from 'react';

import Section from 'components/Section';
import SlotAllocationBar from 'components/SlotAllocationBar';
import Message, { MessageType } from 'shared/components/Message';
import { initClusterOverview, useAgents, useClusterOverview } from 'stores/cluster';
import { ShirtSize } from 'themes';
import { ResourceType } from 'types';
import { getSlotContainerStates } from 'utils/cluster';
import { Loadable } from 'utils/loadable';

export const ClusterOverallBar: React.FC = () => {
  const overview = Loadable.getOrElse(initClusterOverview, useClusterOverview());
  // TODO: handle loading state
  const agents = Loadable.getOrElse([], useAgents());

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
    </Section>
  );
};
