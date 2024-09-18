document.addEventListener('DOMContentLoaded', async () => {
    try {
        const response = await fetch('/api/wallet-info');
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        const data = await response.json();
        document.getElementById('account').innerText = data.account;
        document.getElementById('balance').innerText = data.balance;
        
        const transactions = data.transactions.map(tx => `<li>${tx}</li>`).join('');
        document.getElementById('transactionList').innerHTML = transactions;
    } catch (error) {
        console.error('There was a problem with the fetch operation:', error);
    }
});

document.getElementById('refreshButton').addEventListener('click', async () => {
    // Trigger the fetch call to refresh the wallet info
    document.dispatchEvent(new Event('DOMContentLoaded'));
});
