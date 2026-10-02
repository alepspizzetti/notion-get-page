async () => {
  let root = document.querySelector('.notion-page-content');
  if (!root) return { error: 'page_content_missing' };

  const warnings = [];
  const collapsedSelector = [
    '.notion-toggle-block .notion-list-item-box-left [role="button"][aria-expanded="false"]',
    '.notion-header-block .notion-list-item-box-left [role="button"][aria-expanded="false"]',
    '.notion-sub_header-block .notion-list-item-box-left [role="button"][aria-expanded="false"]',
    '.notion-sub_sub_header-block .notion-list-item-box-left [role="button"][aria-expanded="false"]',
    '.notion-header_4-block .notion-list-item-box-left [role="button"][aria-expanded="false"]',
  ].join(', ');
  let opened = 0;
  for (; opened < 1000; opened++) {
    root = document.querySelector('.notion-page-content');
    const button = root?.querySelector(collapsedSelector);
    if (!button) break;
    const block = button.closest('[data-block-id]');
    const blockId = block?.getAttribute('data-block-id');
    const before = block?.querySelectorAll('[data-block-id]').length || 0;
    button.scrollIntoView({ block: 'center' });
    button.click();
    let lastCount = -1;
    let stable = 0;
    let loaded = before > 0;
    for (let attempt = 0; attempt < 25; attempt++) {
      await new Promise(resolve => setTimeout(resolve, 100));
      const currentRoot = document.querySelector('.notion-page-content');
      const currentBlock = [...(currentRoot?.querySelectorAll('[data-block-id]') || [])]
        .find(candidate => candidate.getAttribute('data-block-id') === blockId);
      const count = currentBlock?.querySelectorAll('[data-block-id]').length || 0;
      loaded ||= count > before;
      stable = count === lastCount ? stable + 1 : 0;
      lastCount = count;
      if (loaded && stable >= 3) break;
    }
    const currentRoot = document.querySelector('.notion-page-content');
    const currentBlock = [...(currentRoot?.querySelectorAll('[data-block-id]') || [])]
      .find(candidate => candidate.getAttribute('data-block-id') === blockId);
    const currentButton = currentBlock?.querySelector('.notion-list-item-box-left [role="button"][aria-expanded]');
    if (currentButton?.getAttribute('aria-expanded') === 'false') {
      warnings.push('Um bloco recolhido não abriu; parte do conteúdo pode estar ausente.');
      break;
    }
    if (!loaded) warnings.push('Um bloco recolhido abriu, mas seu conteúdo não carregou.');
  }
  root = document.querySelector('.notion-page-content');
  if (!root) return { error: 'page_content_missing' };
  if (opened === 1000 || root.querySelector(collapsedSelector))
    warnings.push('Ainda há blocos recolhidos; parte do conteúdo pode estar ausente.');

  const titleNode = document.querySelector('.notion-page-block [data-content-editable-leaf="true"]')
    || document.querySelector('.notion-page-block h1')
    || document.querySelector('h1');
  const title = (titleNode?.textContent || '').trim();
  let markdown = '';
  let source = 'clipboard';
  const escapeMarkdown = text => text.replace(/([\\*_[\]`])/g, '\\$1');
  const inline = node => {
    if (node.nodeType === Node.TEXT_NODE) return escapeMarkdown(node.textContent || '');
    if (node.nodeType !== Node.ELEMENT_NODE) return '';
    const tag = node.tagName.toLowerCase();
    if (tag === 'br') return '\n';
    if (tag === 'img' && node.alt && node.classList.contains('notion-emoji')) return node.alt;
    const text = [...node.childNodes].map(inline).join('');
    if (tag === 'a' && node.href) return `[${text}](${node.href})`;
    if (tag === 'strong' || tag === 'b') return `**${text}**`;
    if (tag === 'em' || tag === 'i') return `*${text}*`;
    if (tag === 's' || tag === 'del' || tag === 'strike') return `~~${text}~~`;
    if (tag === 'code') return `\`${text.replaceAll('`', '\\`')}\``;
    return text;
  };

  // Notion handles the browser copy event and usually places Markdown in text/plain.
  try {
    const marker = `notion-getpage-${crypto.randomUUID()}`;
    await navigator.clipboard.writeText(marker);
    const selection = window.getSelection();
    const range = document.createRange();
    range.selectNodeContents(root);
    selection.removeAllRanges();
    selection.addRange(range);
    document.execCommand('copy');
    await new Promise(resolve => setTimeout(resolve, 100));
    markdown = await navigator.clipboard.readText();
    selection.removeAllRanges();
    if (markdown === marker) markdown = '';
  } catch (_) {
    markdown = '';
  }

  if (!markdown.trim()) {
    source = 'dom';
    warnings.push('O Notion não forneceu Markdown pela cópia; foi usada a página renderizada. Formatação e blocos especiais podem estar incompletos.');

    const blocks = [...root.querySelectorAll('[data-block-id]')];
    const blockSet = new Set(blocks);
    const childrenByParent = new Map();
    for (const block of blocks) {
      let ancestor = block.parentElement;
      while (ancestor && ancestor !== root && !blockSet.has(ancestor)) {
        ancestor = ancestor.parentElement;
      }
      const parent = ancestor && blockSet.has(ancestor) ? ancestor : root;
      if (!childrenByParent.has(parent)) childrenByParent.set(parent, []);
      childrenByParent.get(parent).push(block);
    }
    const leavesByBlock = new Map();
    for (const leaf of root.querySelectorAll('[data-content-editable-leaf="true"]')) {
      const block = leaf.closest('[data-block-id]');
      if (!blockSet.has(block)) continue;
      if (!leavesByBlock.has(block)) leavesByBlock.set(block, []);
      leavesByBlock.get(block).push(leaf);
    }
    const ownLeaves = block => leavesByBlock.get(block) || [];
    const ownText = block => ownLeaves(block).map(leaf => inline(leaf)).join('\n').trim();
    const ownPlainText = block => ownLeaves(block).map(leaf => leaf.textContent || '').join('\n').trim();
    const escapeCell = text => text.replaceAll('|', '\\|').replaceAll('\n', '<br>');
    const typeOf = block => {
      const found = [...block.classList].find(name => /^notion-.+-block$/.test(name));
      return found ? found.slice(7, -6) : 'unknown';
    };
    const seen = new Set();
    const render = (container, depth = 0) => {
      const out = [];
      for (const block of childrenByParent.get(container) || []) {
        const id = block.getAttribute('data-block-id');
        if (seen.has(id)) continue;
        seen.add(id);
        const kind = typeOf(block);
        const text = ownText(block);
        const indent = '  '.repeat(depth);
        let line = '';
        if (kind === 'header') line = `## ${text}`;
        else if (kind === 'sub_header') line = `### ${text}`;
        else if (kind === 'sub_sub_header') line = `#### ${text}`;
        else if (kind === 'header_4') line = `##### ${text}`;
        else if (kind === 'bulleted_list') line = `${indent}- ${text}`;
        else if (kind === 'numbered_list') line = `${indent}1. ${text}`;
        else if (kind === 'to_do') {
          const checked = block.querySelector('[role="checkbox"]')?.getAttribute('aria-checked') === 'true';
          line = `${indent}- [${checked ? 'x' : ' '}] ${text}`;
        } else if (kind === 'toggle') line = `${indent}- ${text}`;
        else if (kind === 'quote') line = text.split('\n').map(part => `> ${part}`).join('\n');
        else if (kind === 'code') {
          const code = ownPlainText(block) || block.querySelector('pre, code')?.textContent.trim() || '';
          if (code) line = `\`\`\`\n${code}\n\`\`\``;
          else warnings.push('Bloco de código sem conteúdo carregado.');
        }
        else if (kind === 'divider') line = '---';
        else if (kind === 'image') {
          const img = block.querySelector('img[src]');
          line = img && !img.src.startsWith('data:') ? `![${img.alt || ''}](${img.src})` : '';
          if (!line) warnings.push('Imagem sem URL utilizável na página renderizada.');
        } else if (kind === 'table') {
          const rows = [...block.querySelectorAll('tr')].map(row =>
            [...row.querySelectorAll('th,td')].map(cell => escapeCell(cell.textContent.trim())));
          if (rows.length) {
            line = rows.map((row, index) => `| ${row.join(' | ')} |${index === 0 ? `\n| ${row.map(() => '---').join(' | ')} |` : ''}`).join('\n');
          }
        } else if (kind === 'page') {
          const link = block.querySelector('a[href]');
          line = link ? `[${text || link.textContent.trim()}](${link.href})` : text;
        } else if (kind === 'video' || kind === 'file' || kind === 'audio' || kind === 'embed') {
          const media = block.querySelector('video[src], audio[src], iframe[src], a[href]');
          if (media) line = `[${text || kind}](${media.src || media.href})`;
          else {
            line = text;
            warnings.push(`Bloco ${kind} sem URL de mídia disponível.`);
          }
        } else if (kind === 'table_of_contents') {
          line = '';
        } else if (kind === 'text' && !text) {
          line = '';
        } else if (text) {
          line = text;
          if (kind !== 'text') warnings.push(`Bloco ${kind} convertido apenas como texto.`);
        } else {
          warnings.push(`Bloco ${kind} sem conteúdo legível na página renderizada.`);
        }
        if (line) out.push(line);
        const children = render(block, depth + (['bulleted_list', 'numbered_list', 'to_do', 'toggle'].includes(kind) ? 1 : 0));
        if (children) out.push(children);
      }
      return out.join('\n\n');
    };
    markdown = render(root);
  }

  markdown = markdown.trim();
  if (title) {
    const first = markdown.split('\n', 1)[0].replace(/^#+\s*/, '').trim();
    if (first === title) {
      const rest = markdown.includes('\n') ? markdown.slice(markdown.indexOf('\n') + 1).trimStart() : '';
      markdown = `# ${title}${rest ? `\n\n${rest}` : ''}`;
    }
    else markdown = `# ${title}${markdown ? `\n\n${markdown}` : ''}`;
  }
  const waitFor = async find => {
    for (let attempt = 0; attempt < 30; attempt++) {
      const found = find();
      if (found) return found;
      await new Promise(resolve => setTimeout(resolve, 100));
    }
    return null;
  };
  const commentsButton = await waitFor(() => [...document.querySelectorAll('[role="button"]')]
    .find(button => /^(Comentários|Comments)$/.test(button.getAttribute('aria-label') || '')));
  if (commentsButton) {
    commentsButton.click();
    const viewAll = await waitFor(() => [...document.querySelectorAll('[role="button"]')]
      .find(button => /^(Ver tudo|View all|See all)$/i.test(button.textContent.trim())));
    if (viewAll) {
      viewAll.click();
      const scroller = await waitFor(() => document.querySelector('.notion-update-sidebar-tab-comments-comments-scroller'));
      if (scroller) {
        const threads = new Map();
        const incomplete = new Set();
        let stable = 0;
        for (let step = 0; step < 100; step++) {
          for (const item of scroller.querySelectorAll('.notion-update-sidebar-tab-comments-discussion-item')) {
            const id = [...item.classList].find(name => name.startsWith('sidebar-discussion-'))?.slice(19);
            if (!id) continue;
            const messages = [...item.querySelectorAll('[aria-label="Ações de comentários"], [aria-label="Comment actions"]')]
              .map(actions => {
                const row = actions.parentElement?.parentElement?.parentElement?.parentElement;
                const body = row?.querySelector('.content-editable-leaf-rtl:not(.discussion-input-text-block)');
                const author = row?.querySelector('[aria-label]')?.getAttribute('aria-label') || 'Autor desconhecido';
                const content = body && inline(body).trim();
                return content ? `- **${escapeMarkdown(author)}:** ${content.replaceAll('\n', '\n  ')}` : '';
              }).filter(Boolean);
            if (messages.length) {
              threads.set(id, messages);
              incomplete.delete(id);
            } else incomplete.add(id);
          }
          const previous = scroller.scrollTop;
          scroller.scrollTop += scroller.clientHeight;
          await new Promise(resolve => setTimeout(resolve, 200));
          stable = scroller.scrollTop === previous ? stable + 1 : 0;
          if (stable === 3) break;
        }
        if (incomplete.size) warnings.push('Uma discussão não forneceu o texto dos comentários.');
        const annotated = [...root.querySelectorAll('[class*="discussion-id-"]')]
          .flatMap(node => [...node.classList].filter(name => name.startsWith('discussion-id-')).map(name => name.slice(14)));
        if (annotated.some(id => !threads.has(id)))
          warnings.push('Alguns comentários marcados na página não apareceram na lista de discussões.');
        if (threads.size) {
          const sections = [...threads].map(([id, messages], index) => {
            const anchor = [...root.querySelectorAll('[class*="discussion-id-"]')]
              .find(node => node.classList.contains(`discussion-id-${id}`));
            const context = anchor?.textContent.trim().replace(/\s+/g, ' ').slice(0, 160);
            return `### ${context ? `Sobre: ${escapeMarkdown(context)}` : `Discussão da página ${index + 1}`}\n\n${messages.join('\n\n')}`;
          });
          markdown = `${markdown.trimEnd()}\n\n## Comentários\n\n${sections.join('\n\n')}`;
        }
      } else warnings.push('O painel de comentários não abriu; os comentários podem estar ausentes.');
    } else if (root.querySelector('[class*="discussion-id-"]')) {
      warnings.push('A lista de comentários não abriu; os comentários podem estar ausentes.');
    }
  } else if (root.querySelector('[class*="discussion-id-"]')) {
    warnings.push('O controle de comentários não apareceu; os comentários podem estar ausentes.');
  }
  if (!markdown) return { error: 'empty_page', warnings };
  return { markdown: `${markdown}\n`, warnings: [...new Set(warnings)], source };
}
