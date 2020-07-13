// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

/**
 *  This method will update styles of continueButton after successful passing the captcha test.
 */
function recaptchaOnSuccess() {
    const continueButton = document.getElementById("captcha-button");
    if (continueButton) {
        continueButton.style.opacity = '1';
        continueButton.style.cursor = 'pointer';
        continueButton.disabled = false;
    }
}

window.onload = () => {
    let token = '';
    const emailInput = document.getElementById('email-input');
    const emailCont = document.getElementById('email-container');
    const captchaForm = document.getElementById('captcha');
    const confirmEmailButton = document.getElementById('submit-email-button');
    const continueButton = document.getElementById("captcha-button");

    emailInput.value = '';

    const button = document.getElementById("mc-embedded-subscribe-footer");

    if (!button) {
        return;
    }

    button.onclick = async function (event) {
        event.preventDefault();

        const email = document.getElementById("mce-EMAIL-newsletter").value;
        const errElm = document.getElementById("mce-error-response-newsletter");
        const succElm = document.getElementById("mce-success-response-newsletter");

        if (!email || !email.includes("@")) {
            errElm.style.display = 'flex';
            succElm.style.display = 'none';

            return;
        }

        const request = {
            method: 'POST',
        };
        let response = await fetch(`/${email}`, request);
        if (!response.ok) {
            return
        }

        errElm.style.display = 'none';
        succElm.style.display = 'flex';
    };

    document.getElementById('logo').addEventListener('click', reload);
    document.getElementById('footer-logo').addEventListener('click', reload);
    document.getElementById('sign-up-btn').addEventListener('click', reload);
    document.getElementById('setup-nav').addEventListener('click', reload);
    document.getElementById('copy-img').addEventListener('click', onCopy);
    document.getElementById('setup-burger-menu').addEventListener('click', toggleAdaptedMenu);
    continueButton.addEventListener('click', continueButtonOnClick);
    confirmEmailButton.addEventListener('click', getToken);
    emailInput.addEventListener('input', buttonState);

    /**
     * should be called after passing reCaptcha checks.
     */
    function continueButtonOnClick() {
        captchaForm.style.display = 'none';
        emailCont.style.display = 'block';
    }

    function buttonState() {
        if (emailInput.value) {
            confirmEmailButton.style.opacity = '1';
            confirmEmailButton.style.cursor = 'pointer';
            confirmEmailButton.disabled = false;

            return;
        }

        confirmEmailButton.style.opacity = '0.3';
        confirmEmailButton.style.cursor = 'default';
        confirmEmailButton.disabled = true;
    }

    function onCopy() {
        const input = document.getElementById('token-input');
        if (input) {
            input.select();
            document.execCommand('copy');
        }
    }

    async function getToken() {
        const value = emailInput.value;
        const rgx = /.*@.*\..*$/;

        if (!rgx.test(value)) {
            return;
        }

        const request = {
            method: 'PUT',
        };
        let response = await fetch(`/${value}`, request);
        if (!response.ok) {
            return;
        }

        token = await response.json();

        toggleSuccess();
    }

    function toggleSuccess() {
        const defaultContainer = document.getElementById('fs-container');
        const successContainer = document.getElementById('fs-success-container');
        const successInput = document.getElementById('token-input');
        if (defaultContainer && successContainer && successInput) {
            defaultContainer.style.display = 'none';
            successContainer.style.display = 'flex';
            successInput.value = token;
        }
    }

    function reload() {
        location.reload();
    }

    function toggleAdaptedMenu() {
        const adaptedMenu = document.getElementById('setup-nav-mobile');
        const burgerMenuIcon = document.getElementById('setup-burger-menu');
        if (adaptedMenu.style.display === 'flex') {
            adaptedMenu.style.display = 'none';
            burgerMenuIcon.style.backgroundColor = 'transparent';

            return;
        }

        adaptedMenu.style.display = 'flex';
        burgerMenuIcon.style.backgroundColor = '#0059D0';
    }
};
