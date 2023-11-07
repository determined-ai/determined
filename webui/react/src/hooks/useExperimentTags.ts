import { TagAction, tagsActionHelper } from 'hew/Tags';
import { useCallback } from 'react';

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
      (experimentId: number, tags: string[]) => {
        const handleTagsChange = async (labels: string[]) => {
          await patchExperiment({ body: { labels }, experimentId });
          if (typeof callbackFn === 'function') {
            callbackFn();
          }
        };
        return tagsActionHelper(tags, handleTagsChange);
      },
      [callbackFn],
    ),
  };
};

export default useExperimentTags;
