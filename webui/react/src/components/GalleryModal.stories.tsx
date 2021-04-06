import React, { useCallback, useState } from 'react';

import GalleryModal from './GalleryModal';

export default {
  component: GalleryModal,
  title: 'GalleryModal',
};

const GALLERY_CONTENT = [
  `
    Twinkle, twinkle, little star
    How I wonder what you are
    Up above the world so high
    Like a diamond in the sky
    Twinkle, twinkle little star
    How I wonder what you are
    
    When the blazing sun is gone
    When he nothing shines upon
    Then you show your little light
    Twinkle, twinkle, all the night
    Twinkle, twinkle, little star
    How I wonder what you are
  `,
  `
    Row, row, row your boat
    Gently down the stream
    Merrily, merrily, merrily, merrily
    Life is but a dream

    Row, row, row your boat
    Gently up the creek If you see a little mouse
    Don’t forget to squeak!

    Row, row, row your boat
    Gently down the stream If you see a crocodile
    Don’t forget to scream!

    Row, row, row your boat
    Gently to the shore
    If you see a lion
    Don’t forget to roar!
  `,
  `
    Humpty Dumpty sat on a wall.
    Humpty Dumpty had a great fall.
    All the king’s horses and all the king’s men
    couldn’t put Humpty together again. (x3)
  `,
];

export const Default = (): React.ReactNode => {
  const [ index, setIndex ] = useState<number>(0);

  const handleNext = useCallback(() => {
    setIndex(prev => {
      return prev === GALLERY_CONTENT.length - 1 ? 0 : prev + 1;
    });
  }, []);

  const handlePrevious = useCallback(() => {
    setIndex(prev => {
      return prev === 0 ? GALLERY_CONTENT.length - 1 : prev - 1;
    });
  }, []);

  return (
    <GalleryModal onNext={handleNext} onPrevious={handlePrevious}>
      <pre>{GALLERY_CONTENT[index]}</pre>
    </GalleryModal>
  );
};
