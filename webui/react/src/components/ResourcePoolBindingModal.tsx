import React, { useRef, useState } from 'react';

import { Modal } from 'components/kit/Modal';
import Transfer from 'components/Transfer';
import handleError from 'utils/error';

import Paragraph from './kit/Typography/Paragraph';

interface Props {
  pool: string;
  bindings: string[];
  workspaces: string[];
  onSave?: (bindings: string[]) => void;
}

const ResourcePoolBindingModalComponent: React.FC<Props> = ({
  pool,
  bindings,
  onSave,
  workspaces,
}: Props) => {
  const bindingList = useRef(bindings).current; // This is only to prevent rerendering
  const [visibleBindings, setVisibleBindings] = useState<string[]>(bindings);

  return (
    <Modal
      cancel
      size="medium"
      submit={{
        handleError,
        handler: async () => {
          await onSave?.(visibleBindings);
        },
        text: 'Apply',
      }}
      title="Manage resource pool bindings">
      <Transfer
        defaultTargetEntries={bindingList}
        entries={workspaces}
        initialTargetEntries={visibleBindings}
        sourceListTitle="Available workspaces"
        targetListTitle={`Bound to ${pool}`}
        onChange={setVisibleBindings}
      />
      <Paragraph>
        Note: Binding a resource pool to a workspace(s) prevents other workspaces from using it.
        Existing running workloads will be unaffected.
      </Paragraph>
    </Modal>
  );
};

export default ResourcePoolBindingModalComponent;
