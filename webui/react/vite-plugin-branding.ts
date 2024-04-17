import type { Plugin } from 'vite';

export const brandHtml = (): Plugin => {
  return {
    name: 'brandHtml',
    async transformIndexHtml(html: string) {
      if (process.env.DET_BUILD_EE === 'true') {
        const cheerio = await import('cheerio');
        const $ = cheerio.load(html);
        $('meta[name="description"]').attr(
          'content',
          'HPE Machine Learning Development Environment',
        );
        return $.html();
      }
      return html;
    },
  };
};
