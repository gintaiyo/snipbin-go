document.addEventListener('DOMContentLoaded', () => {
    const copyBtn = document.getElementById('copy-btn');
    const snippetCode = document.getElementById('snippet-code');

    if (copyBtn && snippetCode) {
        copyBtn.addEventListener('click', async () => {
            try {
                await navigator.clipboard.writeText(snippetCode.innerText);
                const originalText = copyBtn.innerText;
                copyBtn.innerText = 'Copied!';
                copyBtn.style.borderColor = '#238636';

                setTimeout(() => {
                    copyBtn.innerText = originalText;
                    copyBtn.style.borderColor = '';
                }, 2000);
            } catch (err) {
                console.error('Failed to copy: ', err);
            }
        });
    }
});