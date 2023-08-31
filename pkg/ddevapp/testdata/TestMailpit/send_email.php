<?php

// Include the Composer autoloader
require 'vendor/autoload.php';

// Import the PHPMailer class into the global namespace
use PHPMailer\PHPMailer\PHPMailer;

// Create a new PHPMailer instance
$mail = new PHPMailer();

// Set the mail sender
$mail->setFrom('admin@example.com', 'Sender Name');

// Add a recipient
$mail->addAddress('nobody@example.com', 'Nobody at Example');

// Set the subject
$mail->Subject = 'Test using MailPit';

// Set the plain text message body
$mail->Body = 'This is a test of Mailpit in DDEV';

// Send the message
if(!$mail->send()) {
    echo 'Mailer Error: ' . $mail->ErrorInfo;
} else {
    echo 'Message sent!';
}
