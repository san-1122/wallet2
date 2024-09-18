let web3;

async function connectWallet() {
    if (window.ethereum) {
        try {
            await window.ethereum.request({ method: 'eth_requestAccounts' });
            web3 = new Web3(window.ethereum);
            displayMessage("Wallet connected!", "success");
            // Redirect to transfer page
            window.location.href = "/transfer";
        } catch (error) {
            displayMessage("User denied account access", "error");
        }
    } else {
        displayMessage("Please install MetaMask to use this feature.", "error");
    }
}

function displayMessage(message, type) {
    const statusMessage = document.getElementById('statusMessage');
    statusMessage.innerText = message;
    statusMessage.className = type;
    statusMessage.style.display = 'block';
}

document.getElementById('connectButton').addEventListener('click', connectWallet);
