import React, { useState } from 'react';

import Tags, { Props, tagsActionHelper } from 'components/kit/Tags';

export const useTags = (
  tags: string[],
): ((props?: Omit<Props, 'onAction' | 'tags'>) => JSX.Element) => {
  const [ctags, setCtags] = useState<string[]>(tags);
  const onAction = tagsActionHelper(ctags, (ts) => setCtags(ts));
  return (props) => <Tags onAction={onAction} {...props} tags={ctags} />;
};
