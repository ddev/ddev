<?php

ini_set( 'display_errors', 1 );
error_reporting( E_ALL );
$from = "Test Mail <from@mail.test>";
$to = "to@mail.test";
$subject = "PHP Mail Test";
$message = "This is a test to check the PHP Mail functionality";
$headers = "From: " . $from;
$result = mail($to, $subject, $message, $headers);
if ($result == TRUE) {
    echo "Test email sent";
} else {
    echo "Failed to send test email";
}
