import React, { useCallback, useEffect, useState } from 'react';

import TagList from 'components/TagList';
import { getModelVersionLabels } from 'services/api';
import handleError from 'utils/error';

interface Props {
  compact?: boolean;
  disabled?: boolean;
  ghost?: boolean;
  onChange?: (tags: string[]) => void;
  tags: string[];
}

const ModelVersionTagList: React.FC<Props> = ({ disabled = false, ghost, onChange, tags }) => {
  const [ modelVersionTags, setModelVersionTags ] = useState<string[]>();

  const fetchModelVersionLabels = useCallback(async () => {
    try {
      const modelVersionLabels = await getModelVersionLabels({});
      setModelVersionTags(modelVersionLabels);
    } catch (e) {
      handleError(e, { silent: true });
    }
  }, []);

  useEffect(() => {
    fetchModelVersionLabels();
  }, [ fetchModelVersionLabels ]);

  return (
    <TagList
      disabled={disabled}
      ghost={ghost}
      tagCandidates={modelVersionTags}
      tags={tags}
      onChange={onChange}
    />
  );
};

export default ModelVersionTagList;
