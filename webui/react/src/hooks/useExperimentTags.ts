import { useCallback } from 'react';

import { patchExperiment } from 'services/api';

export interface UseExperimentTagsInterface {
  handleTagListChange: (experimentId: number) => (tags: string[]) => void;
}

const useExperimentTags = (callbackFn?: (() => void)): UseExperimentTagsInterface => {
  return {
    handleTagListChange: useCallback((experimentId: number) => async (labels: string[]) => {
      await patchExperiment({ body: { labels }, experimentId });
      if (typeof callbackFn === 'function') {
        callbackFn();
      }
    }, [ callbackFn ]),
  };
};

export default useExperimentTags;
