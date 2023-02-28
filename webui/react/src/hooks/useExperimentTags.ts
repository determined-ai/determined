import { useCallback } from 'react';

import { TagAction } from 'components/kit/Tags';
import { patchExperiment } from 'services/api';

export interface UseExperimentTagsInterface {
  handleTagListChange: (
    experimentId: number,
    tags: string[],
  ) => (action: TagAction, tag: string, updatedId?: number) => void;
}

const useExperimentTags = (callbackFn?: () => void): UseExperimentTagsInterface => {
  return {
    handleTagListChange: useCallback(
      (experimentId: number, tags: string[]) =>
        async (action: TagAction, tag: string, updatedId?: number) => {
          let labels = [...tags];
          if (action === TagAction.Add) {
            labels.push(tag);
          } else if (action === TagAction.Remove) {
            labels = labels.filter((l) => l !== tag);
          } else if (action === TagAction.Update && updatedId !== undefined) {
            labels[updatedId] = tag;
          }
          await patchExperiment({ body: { labels }, experimentId });
          if (typeof callbackFn === 'function') {
            callbackFn();
          }
        },
      [callbackFn],
    ),
  };
};

export default useExperimentTags;
