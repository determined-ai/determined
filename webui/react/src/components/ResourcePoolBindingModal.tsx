import React, { useRef, useState } from 'react';

import { Modal } from 'components/kit/Modal';
import Transfer from 'components/Transfer';
import handleError from 'utils/error';

interface Props {
  pool: string;
  bindings: string[];
  onSave?: (bindings: string[]) => void;
}

const ResourcePoolBindingModalComponent: React.FC<Props> = ({ pool, bindings, onSave }: Props) => {
  const bindingList = useRef(bindings).current; // This is only to prevent rerendering
  const [visibleBindings, setVisibleBindings] = useState<string[]>(bindings);

  return (
    <Modal
      cancel
      size="medium"
      submit={{
        handleError,
        handler: async () => {
          return await onSave?.(visibleBindings);
        },
        text: 'Apply',
      }}
      title="Manage resource pool bindings">
      <Transfer
        defaultTargetEntries={bindingList}
        entries={bindingList}
        initialTargetEntries={visibleBindings}
        sourceListTitle={`Bound to ${pool}`}
        targetListTitle="Available workspaces"
        onChange={setVisibleBindings}
      />
    </Modal>
  );
};

export default ResourcePoolBindingModalComponent;
